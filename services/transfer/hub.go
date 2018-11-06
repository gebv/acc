package transfer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gebv/acca/api/acca"
	"github.com/lib/pq"
)

const cap = 16384

type hub struct {
	db        *sql.DB
	rw        sync.RWMutex
	subs      map[uint]chan<- *acca.Update // subID -> channel
	lastSubID uint
}

func (h *hub) subscribe() (uint, <-chan *acca.Update) {
	ch := make(chan *acca.Update, cap)

	h.rw.Lock()

	h.lastSubID++
	subID := h.lastSubID
	h.subs[subID] = ch

	h.rw.Unlock()

	return subID, ch
}

func (h *hub) reInit() {
	h.rw.Lock()

	for _, ch := range h.subs {
		close(ch)
	}

	h.subs = make(map[uint]chan<- *acca.Update)

	h.rw.Unlock()
}

func (h *hub) unsubscribe(subIDs ...uint) {
	h.rw.Lock()

	for _, subID := range subIDs {
		ch, ok := h.subs[subID]
		if ok {
			delete(h.subs, subID)
			close(ch)
		}
	}

	h.rw.Unlock()
}

func (h *hub) run(ctx context.Context) {
	reportErr := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("Failed to start listener: et=%v, err=%s", ev, err)
		}
	}

	l := pq.NewListener("postgres://acca:acca@127.0.0.1:5432/acca?sslmode=disable", 1*time.Second, 1*time.Second, reportErr)
	defer l.Close()

	h.pgSubscribe(l)

	for ctx.Err() == nil {
		h.reInit()
		h.waitUpdate(l)
	}
}

func (h *hub) waitUpdate(l *pq.Listener) {
	for {
		select {
		case n := <-l.Notify:
			if n == nil {
				// event of reconnected or closed of channel
				return
			}
			u := decodeUpdate(n.Channel, n.Extra)
			log.Printf("Recived msg: %v\n", u.String())
			h.publish(u)
		}
	}
}

func (h *hub) publish(u *acca.Update) {
	h.rw.RLock()
	var toUnsubscribe []uint
	for subID, ch := range h.subs {
		select {
		case ch <- u:

		default:
			toUnsubscribe = append(toUnsubscribe, subID)
		}
	}

	h.rw.RUnlock()

	h.unsubscribe(toUnsubscribe...)
}

func (h *hub) pgSubscribe(l *pq.Listener) {
	err := l.Listen("tx_update_status")
	if err != nil {
		panic(fmt.Sprintf("Failed listen to tx_update_status, err=%v", err))
	}
	log.Println("Listening to tx_update_status")

	err = l.Listen("oper_update_status")
	if err != nil {
		panic(fmt.Sprintf("Failed listen to oper_update_status, err=%v", err))
	}
	log.Println("Listening to oper_update_status")
}

func decodeUpdate(chName, payload string) *acca.Update {
	switch chName {
	case "tx_update_status":
		dto := &acca.Update_TxUpdateStatus{}
		if err := json.Unmarshal([]byte(payload), dto); err != nil {
			panic(fmt.Sprintln("Failed decode DTO for 'tx_update_status': ", err))
		}
		return &acca.Update{
			Type: &acca.Update_TxStatus{
				TxStatus: dto,
			},
		}
	case "oper_update_status":
		dto := &acca.Update_OperUpdateStatus{}
		if err := json.Unmarshal([]byte(payload), dto); err != nil {
			panic(fmt.Sprintln("Failed decode DTO for 'oper_update_status': ", err))
		}
		return &acca.Update{
			Type: &acca.Update_OperStatus{
				OperStatus: dto,
			},
		}
	default:
		panic(fmt.Sprintln("Failed build update DTO - not exptected event name: ", chName))
	}
}

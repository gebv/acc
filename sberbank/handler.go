package sberbank

import "net/http"

type Webhook struct {
}

func (w *Webhook) ServeHTTP(http.ResponseWriter, *http.Request) {

}

var _ http.Handler = (*Webhook)(nil)

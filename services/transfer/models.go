package transfer

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/gebv/acca/api/acca"
)

type Meta map[string]string

func MetaFrom(v map[string]string) Meta {
	return Meta(v)
}

func (p Meta) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Meta) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		return json.NewDecoder(buf).Decode(p)
	case string:
		buf := bytes.NewBufferString(v)
		return json.NewDecoder(buf).Decode(p)
	default:
		return fmt.Errorf("accounts.Meta: not expected type %T", in)
	}
}

type pgOpers []*acca.TxOper

func (ts pgOpers) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(ts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var (
	mapToApiOperStatus = map[string]acca.OperStatus{
		"unknown":  acca.OperStatus_OPER_UNKNOWN,
		"draft":    acca.OperStatus_OPER_DRAFT,
		"hold":     acca.OperStatus_OPER_HOLD,
		"accepted": acca.OperStatus_OPER_ACCEPTED,
		"rejected": acca.OperStatus_OPER_REJECTED,
	}
	mapToApiTxStatus = map[string]acca.TxStatus{
		"unknown":  acca.TxStatus_UNKNOWN_TS,
		"draft":    acca.TxStatus_DRAFT,
		"uth":      acca.TxStatus_AUTH,
		"accepted": acca.TxStatus_ACCEPTED,
		"rejected": acca.TxStatus_REJECTED,
		"ailed":    acca.TxStatus_FAILED,
	}
)

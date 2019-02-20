package accounts

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

type BalancesShortInfo []*acca.BalanceShortInfo

func (p BalancesShortInfo) Value() (driver.Value, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *BalancesShortInfo) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		return json.NewDecoder(buf).Decode(p)
	case string:
		buf := bytes.NewBufferString(v)
		return json.NewDecoder(buf).Decode(p)
	case nil:
		return nil
	default:
		return fmt.Errorf("accounts.BalancesShortInfo: not expected type %T", in)
	}
}

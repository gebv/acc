// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: api/accounts.proto

package api

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/ptypes/wrappers"
	github_com_mwitkow_go_proto_validators "github.com/mwitkow/go-proto-validators"
	math "math"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

func (this *CreateCurrencyRequest) Validate() error {
	if this.Meta != nil {
		if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(this.Meta); err != nil {
			return github_com_mwitkow_go_proto_validators.FieldError("Meta", err)
		}
	}
	return nil
}
func (this *CreateCurrencyResponse) Validate() error {
	return nil
}
func (this *GetCurrencyRequest) Validate() error {
	return nil
}
func (this *GetCurrencyResponse) Validate() error {
	if this.Currency != nil {
		if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(this.Currency); err != nil {
			return github_com_mwitkow_go_proto_validators.FieldError("Currency", err)
		}
	}
	return nil
}
func (this *CreateAccountRequest) Validate() error {
	if this.Meta != nil {
		if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(this.Meta); err != nil {
			return github_com_mwitkow_go_proto_validators.FieldError("Meta", err)
		}
	}
	return nil
}
func (this *CreateAccountResponse) Validate() error {
	return nil
}
func (this *GetAccountByKeyRequest) Validate() error {
	return nil
}
func (this *GetAccountByKeyResponse) Validate() error {
	if this.Account != nil {
		if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(this.Account); err != nil {
			return github_com_mwitkow_go_proto_validators.FieldError("Account", err)
		}
	}
	return nil
}
func (this *BalanceChangesRequest) Validate() error {
	return nil
}
func (this *BalanceChangesResponse) Validate() error {
	for _, item := range this.BalanceChanges {
		if item != nil {
			if err := github_com_mwitkow_go_proto_validators.CallValidatorIfExists(item); err != nil {
				return github_com_mwitkow_go_proto_validators.FieldError("BalanceChanges", err)
			}
		}
	}
	return nil
}

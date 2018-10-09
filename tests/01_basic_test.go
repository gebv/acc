package tests

import (
	"fmt"
	"testing"
)

type accountInfo1 struct {
	AccID   string
	Balance uint64
}

type testCase1 struct {
	CaseName string

	InitAccounts []accountInfo1

	Transfers transfers
	MetaData  MetaData
	Reason    string

	TxID             int64
	ExpectedBalances map[string]uint64

	TxStatusBefore string
	TxStatusAfter  string

	NumProcess int
}

func Test01Basic_01SimpleTrasnferWithoutHold(t *testing.T) {
	prefix := "test01.01."
	cur := prefix + "curr"
	accID := func(accID string) string {
		return prefix + accID
	}

	tests := []cmdBatch{
		{
			"InternalTransfer",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("1.1"),
						Balance: 10,
					},
					{
						AccID:   accID("1.2"),
						Balance: 20,
					},
					{
						AccID:   accID("1.3"),
						Balance: 30,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAccID: accID("1.1"),
							DstAccID: accID("1.2"),
							Type:     Internal,
							Amount:   9,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
						{
							SrcAccID: accID("1.2"),
							DstAccID: accID("1.3"),
							Type:     Internal,
							Amount:   29,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("accepted"),
				CmdCheckBalances(map[string]uint64{
					accID("1.1"): 10 - 9,
					accID("1.2"): 20 + 9 - 29,
					accID("1.3"): 30 + 29,
				}),
			},
		},
		{
			"InternalTransfer_Error_NotEnoughMoney",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("2.1"),
						Balance: 10,
					},
					{
						AccID:   accID("2.2"),
						Balance: 20,
					},
					{
						AccID:   accID("2.3"),
						Balance: 30,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAccID: accID("2.1"),
							DstAccID: accID("2.2"),
							Type:     Internal,
							Amount:   1,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
						{
							SrcAccID: accID("2.2"),
							DstAccID: accID("2.3"),
							Type:     Internal,
							Amount:   1000,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("failed"),
				CmdCheckBalances(map[string]uint64{
					accID("2.1"): 10,
					accID("2.2"): 20,
					accID("2.3"): 30,
				}),
			},
		},
		{
			"InternalTransfer_EmptyListTransfers",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("3.1"),
						Balance: 10,
					},
					{
						AccID:   accID("3.2"),
						Balance: 20,
					},
					{
						AccID:   accID("3.3"),
						Balance: 30,
					},
				}),
				CmdTransfers([]transfers{
					transfers{}, // empty list transaction
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("accepted"),
				CmdCheckBalances(map[string]uint64{
					accID("3.1"): 10,
					accID("3.2"): 20,
					accID("3.3"): 30,
				}),
			},
		},

		{
			"InternalTransfer_Recharge",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("4.payment_gateway"),
						Balance: 0,
					},
					{
						AccID:   accID("4.client"),
						Balance: 0,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAccID: accID("4.payment_gateway"),
							DstAccID: accID("4.client"),
							Type:     Recharge,
							Amount:   102,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("accepted"),
				CmdCheckBalances(map[string]uint64{
					accID("4.payment_gateway"): 102,
					accID("4.client"):          102,
				}),
			},
		},

		{
			"InternalTransfer_Withdraw",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("5.payment_gateway"),
						Balance: 100,
					},
					{
						AccID:   accID("5.client"),
						Balance: 10,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAccID: accID("5.client"),
							DstAccID: accID("5.payment_gateway"),
							Type:     Withdraw,
							Amount:   10,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("accepted"),
				CmdCheckBalances(map[string]uint64{
					accID("5.payment_gateway"): 90,
					accID("5.client"):          0,
				}),
			},
		},

		{
			"InternalTransfer_WithdrawError1_NotEnoughMoneyFromSysAccount",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("6.payment_gateway"),
						Balance: 0,
					},
					{
						AccID:   accID("6.client"),
						Balance: 10,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAccID: accID("6.client"),
							DstAccID: accID("6.payment_gateway"),
							Type:     Withdraw,
							Amount:   10,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("failed"),
				CmdCheckBalances(map[string]uint64{
					accID("6.payment_gateway"): 0,
					accID("6.client"):          10,
				}),
			},
		},
	}

	for _, tt := range tests {
		txIDs := []int64{}
		for index, cmd := range tt.Commands {
			t.Run(tt.Name+"_#"+fmt.Sprint(index+1), func(t *testing.T) {
				cmdApply(t, cmd, &txIDs)
			})
		}
		t.Run(tt.Name+"_#Destroy", func(t *testing.T) {
			t.Logf("Tx IDs: %+v", txIDs)
		})
	}
}
func Test01Basic_02SimpleTransferWithHold(t *testing.T) {
	prefix := "test01.02."
	cur := prefix + "curr"
	accID := func(accID string) string {
		return prefix + accID
	}

	tests := []cmdBatch{
		{
			"InternalTransferWithHold",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccID:   accID("1"),
						Balance: 10,
					},
					{
						AccID:   accID("2"),
						Balance: 20,
					},
					{
						AccID:   accID("3"),
						Balance: 30,
					},
					{
						AccID:   accID("hold1"),
						Balance: 0,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAccID: accID("1"),
							DstAccID: accID("2"),
							Type:     Internal,
							Amount:   9,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold:      true,
							HoldAccID: accID("hold1"),
						},
						{
							SrcAccID: accID("2"),
							DstAccID: accID("3"),
							Type:     Internal,
							Amount:   19,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold:      true,
							HoldAccID: accID("hold1"),
						},
						{
							SrcAccID: accID("3"),
							DstAccID: accID("1"),
							Type:     Internal,
							Amount:   29,
							Reason:   "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold:      true,
							HoldAccID: accID("hold1"),
						},
					},
				}),
				CmdCheckStatuses("draft"),
				CmdExecute(1),
				CmdCheckStatuses("auth"),
				CmdCheckBalances(map[string]uint64{
					accID("1"):     10 - 9,
					accID("2"):     20 - 19,
					accID("3"):     30 - 29,
					accID("hold1"): 9 + 19 + 29,
				}),
				CmdApprove(0),
				CmdExecute(1),
				CmdCheckStatuses("accepted"),
				CmdCheckBalances(map[string]uint64{
					accID("1"):     10 - 9 + 29,
					accID("2"):     20 - 19 + 9,
					accID("3"):     30 - 29 + 19,
					accID("hold1"): 0,
				}),
			},
		},
	}

	for _, tt := range tests {
		txIDs := []int64{}
		for index, cmd := range tt.Commands {
			t.Run(tt.Name+"_#"+fmt.Sprint(index+1), func(t *testing.T) {
				cmdApply(t, cmd, &txIDs)
			})
		}
		t.Run(tt.Name+"_#Destroy", func(t *testing.T) {
			t.Logf("Tx IDs: %+v", txIDs)
		})
	}
}

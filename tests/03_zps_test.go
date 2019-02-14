package tests

// import "testing"

// func Test03Basic_ZPSCases(t *testing.T) {
// 	cur := "curr"

// 	tests := []cmdBatch{
// 		{
// 			"InternalTransfer_Recharge",
// 			[]command{
// 				CmdInitAccounts(cur, []accountInfo{
// 					{
// 						AccKey:  "payment_gateway",
// 						Balance: 0,
// 					},
// 					{
// 						AccKey:  "sold_buff",
// 						Balance: 0,
// 					},
// 					{
// 						AccKey:  "client",
// 						Balance: 0,
// 					},
// 					{
// 						AccKey:  "hold1",
// 						Balance: 0,
// 					},
// 				}),
// 				CmdTransfers([]transfers{
// 					transfers{
// 						{
// 							SrcAcc: "payment_gateway",
// 							DstAcc: "client",
// 							Type:   Recharge,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold: false,
// 						},
// 					},

// 					transfers{
// 						{
// 							SrcAcc: "payment_gateway",
// 							DstAcc: "client",
// 							Type:   Recharge,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold:    true,
// 							HoldAcc: "hold1",
// 						},
// 						{
// 							SrcAcc: "client",
// 							DstAcc: "sold_buff",
// 							Type:   Internal,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold:    true,
// 							HoldAcc: "hold1",
// 						},
// 					},

// 					transfers{
// 						{
// 							SrcAcc: "payment_gateway",
// 							DstAcc: "client",
// 							Type:   Recharge,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold:    true,
// 							HoldAcc: "hold1",
// 						},
// 						{
// 							SrcAcc: "client",
// 							DstAcc: "sold_buff",
// 							Type:   Internal,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold:    true,
// 							HoldAcc: "hold1",
// 						},
// 					},

// 					transfers{
// 						{
// 							SrcAcc: "payment_gateway",
// 							DstAcc: "client",
// 							Type:   Recharge,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold:    true,
// 							HoldAcc: "hold1",
// 						},
// 						{
// 							SrcAcc: "client",
// 							DstAcc: "sold_buff",
// 							Type:   Internal,
// 							Amount: 102,
// 							Reason: "fortesting",
// 							Meta: MetaData{
// 								"foo": "bar",
// 							},
// 							Hold:    true,
// 							HoldAcc: "hold1",
// 						},
// 					},
// 				}),
// 				CmdCheckStatuses("draft", "draft", "draft", "draft"),
// 				CmdExecute(4),
// 				CmdCheckStatuses("accepted", "auth", "auth", "auth"),
// 				// CmdApprove(0),
// 				// CmdExecute(1),
// 				// CmdCheckStatuses("accepted", "auth"),
// 			},
// 		},
// 	}

// 	runTests(t, tests)
// }

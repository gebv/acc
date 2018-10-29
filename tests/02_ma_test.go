package tests

import "testing"

func Test02MA_01Basic(t *testing.T) {
	// TODO:
	// - создать счета у которых префикс относится к пользовтаелю а счета с разными типами
	// - добавить новую вьюшку в которой учитываются пользовательские аккаунты и верно их суммирует

	cur := "curr"
	user1 := "ma.u1."
	user2 := "ma.u2."

	tests := []cmdBatch{
		{
			"InternalTransferWithHold",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccKey:  user1 + "main",
						Balance: 10,
					},
					{
						AccKey:  user1 + "bonus",
						Balance: 1000,
					},
					{
						AccKey:  user1 + "credit",
						Balance: 100,
					}, // 1000-100+10 = 910

					{
						AccKey:  user2 + "main",
						Balance: 10,
					},
					{
						AccKey:  user2 + "bonus",
						Balance: 1000,
					},
					{
						AccKey:  user2 + "credit",
						Balance: 100,
					}, // 1000-100+10 = 910
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAcc: user1 + "bonus",
							DstAcc: user2 + "bonus",
							Type:   Internal,
							Amount: 9,
							Reason: "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
						{
							SrcAcc: user2 + "credit",
							DstAcc: user1 + "credit",
							Type:   Internal,
							Amount: 1,
							Reason: "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
					transfers{
						{
							SrcAcc: user1 + "bonus",
							DstAcc: user2 + "bonus",
							Type:   Internal,
							Amount: 9,
							Reason: "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
						{
							SrcAcc: user2 + "credit",
							DstAcc: user1 + "credit",
							Type:   Internal,
							Amount: 1,
							Reason: "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold: false,
						},
					},
				}),
				CmdCheckStatuses("draft", "draft"),
				CmdExecute(2),
				CmdCheckStatuses("accepted", "accepted"),
				CmdCheckBalances(map[string]uint64{
					user1 + "bonus": 1000 - 9 - 9,
					user2 + "bonus": 1000 + 9 + 9,

					user2 + "credit": 100 - 1 - 1,
					user1 + "credit": 100 + 1 + 1,
				}),

				// TODO: check total
				// user1 = 910 + 1 + 1 - 9 - 9 = 894
				// user2 = 910 - 1 - 1 + 9 + 9 = 926
			},
		},
	}

	runTests(t, tests)
}

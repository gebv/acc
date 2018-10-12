package tests

import (
	"testing"
)

func ExampleTest(t *testing.T) {
	cur := "curr"

	tests := []cmdBatch{
		{
			"ExampleTransfer",
			[]command{
				CmdInitAccounts(cur, []accountInfo{
					{
						AccKey:  "1",
						Balance: 10,
					},
					{
						AccKey:  "2",
						Balance: 20,
					},
					{
						AccKey:  "hold1",
						Balance: 0,
					},
				}),
				CmdTransfers([]transfers{
					transfers{
						{
							SrcAcc: "1",
							DstAcc: "2",
							Type:   Internal,
							Amount: 9,
							Reason: "fortesting",
							Meta: MetaData{
								"foo": "bar",
							},
							Hold:    true,
							HoldAcc: "hold1",
						},
					},
				}),
			},
		},
	}

	runTests(t, tests)
}

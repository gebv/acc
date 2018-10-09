package tests

import (
	"fmt"
	"testing"
)

func ExampleTest(t *testing.T) {
	prefix := "test###.###."
	cur := prefix + "curr"
	accID := func(accID string) string {
		return prefix + accID
	}

	tests := []cmdBatch{
		{
			"ExampleTransfer",
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
					},
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

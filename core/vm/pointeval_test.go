package vm

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func FuzzPointEvaluation(f *testing.F) {
	p := &pointEvaluation{}
	f.Add(common.FromHex("013c03613f6fc558fb7e61e75602241ed9a2f04e36d8670aadd286e71b5ca9cc420000000000000000000000000000000000000000000000000000000000000031e5a2356cbc2ef6a733eae8d54bf48719ae3d990017ca787c419c7d369f8e3c83fac17c3f237fc51f90e2c660eb202a438bc2025baded5cd193c1a018c5885bc9281ba704d5566082e851235c7be763b2a99adff965e0a121ee972ebc472d02944a74f5c6243e14052e105124b70bf65faf85ad3a494325e269fad097842cba"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 192 {
			return
		}
		if len(data) > 512 {
			return
		}
		p.Run(data[:192])
	})

}

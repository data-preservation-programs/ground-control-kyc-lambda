package minpower

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	in   string
	want bool
}

func TestMinPower(t *testing.T) {
	min, ok := new(big.Int).SetString("10995116277760", 10) // 10TiB = 10 * 1024^4
	assert.True(t, ok)

	minerID := os.Getenv("MINER_ID")

	cases := make([]TestCase, 0)
	if minerID == "" {
		cases = append(cases, TestCase{"f01000", false})
		cases = append(cases, TestCase{"f02620", true})
	} else {
		cases = append(cases, TestCase{minerID, true})
	}
	for _, c := range cases {
		ok, err := MinQualityPowerOk(context.Background(), c.in, min)
		assert.Equal(t, c.want, ok)
		assert.Nil(t, err)
	}
}

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func LambdaTest(t *testing.T) {
	ctx := context.Background()
	assert.True(t, ctx != nil)
}

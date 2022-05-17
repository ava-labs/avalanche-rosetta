// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package transfer

import (
	"github.com/onsi/gomega"

	"github.com/ava-labs/avalanche-rosetta/tests/e2e"

	ginkgo "github.com/onsi/ginkgo/v2"
)

var _ = ginkgo.Describe("[Local] Testing", func() {
	ginkgo.It("nits", func() {
		uris := e2e.GetURIs()
		gomega.Expect(uris).ShouldNot(gomega.BeEmpty())
	})
})

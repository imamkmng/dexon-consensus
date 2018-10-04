// Copyright 2018 The dexon-consensus-core Authors
// This file is part of the dexon-consensus-core library.
//
// The dexon-consensus-core library is free software: you can redistribute it
// and/or modify it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The dexon-consensus-core library is distributed in the hope that it will be
// useful, but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the dexon-consensus-core library. If not, see
// <http://www.gnu.org/licenses/>.

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/dexon-foundation/dexon-consensus-core/common"
	"github.com/dexon-foundation/dexon-consensus-core/core/types"
)

type slowApp struct {
	sleep                  time.Duration
	blockConfirmed         map[common.Hash]struct{}
	stronglyAcked          map[common.Hash]struct{}
	totalOrderingDelivered map[common.Hash]struct{}
	blockDelivered         map[common.Hash]struct{}
}

func newSlowApp(sleep time.Duration) *slowApp {
	return &slowApp{
		sleep:                  sleep,
		blockConfirmed:         make(map[common.Hash]struct{}),
		stronglyAcked:          make(map[common.Hash]struct{}),
		totalOrderingDelivered: make(map[common.Hash]struct{}),
		blockDelivered:         make(map[common.Hash]struct{}),
	}
}

func (app *slowApp) PreparePayload(_ types.Position) []byte {
	return []byte{}
}

func (app *slowApp) PrepareWitness(_ uint64) types.Witness {
	return types.Witness{}
}

func (app *slowApp) VerifyBlock(_ *types.Block) bool {
	return true
}

func (app *slowApp) BlockConfirmed(blockHash common.Hash) {
	time.Sleep(app.sleep)
	app.blockConfirmed[blockHash] = struct{}{}
}

func (app *slowApp) StronglyAcked(blockHash common.Hash) {
	time.Sleep(app.sleep)
	app.stronglyAcked[blockHash] = struct{}{}
}

func (app *slowApp) TotalOrderingDelivered(blockHashes common.Hashes, early bool) {
	time.Sleep(app.sleep)
	for _, hash := range blockHashes {
		app.totalOrderingDelivered[hash] = struct{}{}
	}
}

func (app *slowApp) BlockDelivered(block types.Block) {
	time.Sleep(app.sleep)
	app.blockDelivered[block.Hash] = struct{}{}
}

type NonBlockingTestSuite struct {
	suite.Suite
}

func (s *NonBlockingTestSuite) TestNonBlocking() {
	sleep := 50 * time.Millisecond
	app := newSlowApp(sleep)
	nbModule := newNonBlocking(app, app)
	hashes := make(common.Hashes, 10)
	for idx := range hashes {
		hashes[idx] = common.NewRandomHash()
	}
	now := time.Now().UTC()
	shouldFinish := now.Add(100 * time.Millisecond)

	// Start doing some 'heavy' job.
	for _, hash := range hashes {
		nbModule.BlockConfirmed(hash)
		nbModule.StronglyAcked(hash)
		nbModule.BlockDelivered(types.Block{
			Hash:    hash,
			Witness: types.Witness{Timestamp: time.Now().UTC()},
		})
	}
	nbModule.TotalOrderingDelivered(hashes, true)

	// nonBlocking should be non-blocking.
	s.True(shouldFinish.After(time.Now().UTC()))

	nbModule.wait()
	for _, hash := range hashes {
		s.Contains(app.blockConfirmed, hash)
		s.Contains(app.stronglyAcked, hash)
		s.Contains(app.totalOrderingDelivered, hash)
		s.Contains(app.blockDelivered, hash)
	}
}

func TestNonBlocking(t *testing.T) {
	suite.Run(t, new(NonBlockingTestSuite))
}
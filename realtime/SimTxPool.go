package realtime

import (
	// "fmt"
	// "time"

	// "github.com/ethereum/go-ethereum/consensus"
	// "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	// "os"
	// "math/big"
	// "github.com/ethereum/go-ethereum/trace"
	// "github.com/ethereum/go-ethereum/eth/downloader"
)



var (
	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	SimErrAlreadyKnown = errors.New("already known")

	// ErrAlreadyKnown is returned if the transactions is already contained
	// within the pool.
	SimErrAlreadyExecuted = errors.New("already executed")

	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	SimErrInvalidSender = errors.New("invalid sender")

	// ErrUnderpriced is returned if a transaction's gas price is below the minimum
	// configured for the transaction pool.
	SimErrUnderpriced = errors.New("transaction underpriced")

	// ErrTxPoolOverflow is returned if the transaction pool is full and can't accpet
	// another remote transaction.
	SimErrTxPoolOverflow = errors.New("txpool is full")

	// ErrReplaceUnderpriced is returned if a transaction is attempted to be replaced
	// with a different one without the required price bump.
	SimErrReplaceUnderpriced = errors.New("replacement transaction underpriced")

	// ErrGasLimit is returned if a transaction's requested gas limit exceeds the
	// maximum allowance of the current block.
	SimErrGasLimit = errors.New("exceeds block gas limit")

	// ErrNegativeValue is a sanity error to ensure no one is able to specify a
	// transaction with a negative value.
	SimErrNegativeValue = errors.New("negative value")

	// ErrOversizedData is returned if the input data of a transaction is greater
	// than some meaningful limit a user might use. This is not a consensus error
	// making the transaction invalid, rather a DOS protection.
	SimErrOversizedData = errors.New("oversized data")
)


type SimTxPool struct {
	// general info
	// config      TxPoolConfig
	chainconfig *params.ChainConfig
	chain       blockChain
	signer      types.Signer


	// pending transactions are ones to be executed
	pending  map[common.Hash]*types.Transaction
	// queue transactions have not reached requirements, like the message nonce, 
	queue  map[common.Hash]*types.Transaction
	// already been executed?
	// In this level, excludes the transactions have been executed
	executed map[common.Hash]*types.Transaction
	// all the tx in pool
	// find whether in the pending or the queue, or the executed
	all     *txLookup                    // All transactions to allow lookup

	// add pool
	mu          sync.RWMutex

	// Current gas limit for transaction caps
	currentMaxGas uint64         
}



func NewSimTxPool() *SimTxPool {
	// Create the transaction pool with its initial settings
	pool := &SimTxPool{
		// config:          config,
		chainconfig:     chainconfig,
		chain:           chain,
		signer:          types.NewEIP155Signer(chainconfig.ChainID),

		pending:         make(map[common.Address]*txList),
		queue:           make(map[common.Address]*txList),
		executed: 		 make(map[common.Address]*txList),
		all:             newTxLookup(),

		currentMaxGas:   chain.CurrentBlock().Header().GasLimit, 
	}

	return pool
}


const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10

	// txSlotSize is used to calculate how many data slots a single transaction
	// takes up based on its size. The slots are used as DoS protection, ensuring
	// that validating a new transaction remains a constant operation (in reality
	// O(maxslots), where max slots are 4 currently).
	txSlotSize = 32 * 1024

	// txMaxSize is the maximum size a single transaction can have. This field has
	// non-trivial consequences: larger transactions are significantly harder and
	// more expensive to propagate; larger transactions also take more resources
	// to validate whether they fit into the pool or not.
	txMaxSize = 4 * txSlotSize // 128KB
)



// validateTx checks whether a transaction is valid according to the consensus
// rules and adheres to some heuristic limits of the local node (price and size).
func (pool *SimTxPool) validateTx(tx *types.Transaction) error {
	// Reject transactions over defined size to prevent DOS attacks
	if uint64(tx.Size()) > txMaxSize {
		return SimErrOversizedData
	}
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return SimErrNegativeValue
	}
	// Ensure the transaction doesn't exceed the current block limit gas.
	if pool.currentMaxGas < tx.Gas() {
		return SimErrGasLimit
	}
	// Make sure the transaction is signed properly
	from, err := types.Sender(pool.signer, tx)
	if err != nil {
		return SimErrInvalidSender
	}
	// // Drop non-local transactions under our own minimal accepted gas price
	// if !local && tx.GasPriceIntCmp(pool.gasPrice) < 0 {
	// 	return ErrUnderpriced
	// }

	// Ensure the transaction adheres to nonce ordering
	if pool.currentState.GetNonce(from) > tx.Nonce() {
		return ErrNonceTooLow
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if pool.currentState.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}
	// Ensure the transaction has more gas than the basic tx fee.
	intrGas, err := IntrinsicGas(tx.Data(), tx.To() == nil, true, pool.istanbul)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}
	return nil
}



// addTxsLocked attempts to queue a batch of transactions if they are valid.
// The transaction pool lock must be held.
func (pool *SimTxPool) addTxsLocked(txs []*types.Transaction) []error {
	errs := make([]error, len(txs))
	for i, tx := range txs {
		// already validated
		from, _ := types.Sender(pool.signer, tx)
		current_state := pool.chain.StateAt(chain.CurrentBlock().Root())
		// should be listed in the queue?
		if current_state.GetNonce(from) < tx.Nonce() {
			pool.queue[tx.Hash()] = tx
			pool.all.Add(tx, false)
			continue
		}
		pool.pending[tx.Hash()] = tx
		pool.all.Add(tx, true)
		errs[i] = err
	}
	return errs
}




// useless now
// add validates a transaction and inserts it into the non-executable queue for later
// pending promotion and execution. If the transaction is a replacement for an already
// pending or queued one, it overwrites the previous transaction if its price is higher.
//
// If a newly added transaction is marked as local, its sending account will be
// whitelisted, preventing any associated transaction from being dropped out of the pool
// due to pricing constraints.
func (pool *SimTxPool) add(tx *types.Transaction) (replaced bool, err error) {
	// Already done before 
	// If the transaction is already known, discard it
	// hash := tx.Hash()
	// if pool.all.Get(hash) != nil {
	// 	log.Trace("Discarding already known transaction", "hash", hash)
	// 	knownTxMeter.Mark(1)
	// 	return false, ErrAlreadyKnown
	// }
	// // Make the local flag. If it's from local source or it's from the network but
	// // the sender is marked as local previously, treat it as the local transaction.
	// isLocal := local || pool.locals.containsTx(tx)

	// // If the transaction fails basic validation, discard it
	// if err := pool.validateTx(tx, isLocal); err != nil {
	// 	log.Trace("Discarding invalid transaction", "hash", hash, "err", err)
	// 	invalidTxMeter.Mark(1)
	// 	return false, err
	// }


	// Mia: have not considered this
	// // If the transaction pool is full, discard underpriced transactions
	// if uint64(pool.all.Count()+numSlots(tx)) > pool.config.GlobalSlots+pool.config.GlobalQueue {
	// 	// If the new transaction is underpriced, don't accept it
	// 	if !isLocal && pool.priced.Underpriced(tx) {
	// 		log.Trace("Discarding underpriced transaction", "hash", hash, "price", tx.GasPrice())
	// 		underpricedTxMeter.Mark(1)
	// 		return false, ErrUnderpriced
	// 	}
	// 	// New transaction is better than our worse ones, make room for it.
	// 	// If it's a local transaction, forcibly discard all available transactions.
	// 	// Otherwise if we can't make enough room for new one, abort the operation.
	// 	drop, success := pool.priced.Discard(pool.all.Slots()-int(pool.config.GlobalSlots+pool.config.GlobalQueue)+numSlots(tx), isLocal)

	// 	// Special case, we still can't make the room for the new remote one.
	// 	if !isLocal && !success {
	// 		log.Trace("Discarding overflown transaction", "hash", hash)
	// 		overflowedTxMeter.Mark(1)
	// 		return false, ErrTxPoolOverflow
	// 	}
	// 	// Kick out the underpriced remote transactions.
	// 	for _, tx := range drop {
	// 		log.Trace("Discarding freshly underpriced transaction", "hash", tx.Hash(), "price", tx.GasPrice())
	// 		underpricedTxMeter.Mark(1)
	// 		pool.removeTx(tx.Hash(), false)
	// 	}


	// have not figured out the 
	// Try to replace an existing transaction in the pending pool
	// from, _ := types.Sender(pool.signer, tx) // already validated
	// if list := pool.pending[from]; list != nil && list.Overlaps(tx) {
	// 	// Nonce already pending, check if required price bump is met
	// 	inserted, old := list.Add(tx, pool.config.PriceBump)
	// 	if !inserted {
	// 		pendingDiscardMeter.Mark(1)
	// 		return false, ErrReplaceUnderpriced
	// 	}
	// 	// New transaction is better, replace old one
	// 	if old != nil {
	// 		pool.all.Remove(old.Hash())
	// 		pool.priced.Removed(1)
	// 		pendingReplaceMeter.Mark(1)
	// 	}
	// 	pool.all.Add(tx, isLocal)
	// 	pool.priced.Put(tx, isLocal)
	// 	pool.journalTx(from, tx)
	// 	pool.queueTxEvent(tx)
	// 	log.Trace("Pooled new executable transaction", "hash", hash, "from", from, "to", tx.To())

	// 	// Successful promotion, bump the heartbeat
	// 	pool.beats[from] = time.Now()
	// 	return old != nil, nil
	// }
	// // New transaction isn't replacing a pending one, push into queue
	// replaced, err = pool.enqueueTx(hash, tx, isLocal, true)
	// if err != nil {
	// 	return false, err
	// }
	// // Mark local addresses and journal local transactions
	// if local && !pool.locals.contains(from) {
	// 	log.Info("Setting new local account", "address", from)
	// 	pool.locals.add(from)
	// 	pool.priced.Removed(pool.all.RemoteToLocals(pool.locals)) // Migrate the remotes if it's marked as local first time.
	// }
	// if isLocal {
	// 	localGauge.Inc(1)
	// }
	// pool.journalTx(from, tx)

	// log.Trace("Pooled new future transaction", "hash", hash, "from", from, "to", tx.To())
	// return replaced, nil
}


// txLookup is used internally by TxPool to track transactions while allowing
// lookup without mutex contention.
//
// Note, although this type is properly protected against concurrent access, it
// is **not** a type that should ever be mutated or even exposed outside of the
// transaction pool, since its internal state is tightly coupled with the pools
// internal mechanisms. The sole purpose of the type is to permit out-of-bound
// peeking into the pool in TxPool.Get without having to acquire the widely scoped
// TxPool.mu mutex.
type txLookup struct {
	// Mia add: do not understand the slots
	// slots   int
	lock    sync.RWMutex
	pending  map[common.Hash]*types.Transaction
	queue map[common.Hash]*types.Transaction
}

type TxStatus uint

const (
	TxStatusUnknown TxStatus = iota
	TxStatusQueued
	TxStatusPending
	// TxStatusIncluded
	// TxStatusExecuted

)

// Get returns a transaction if it exists in the lookup, or nil if not found.
func (t *txLookup) Get(hash common.Hash) *types.Transaction {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if tx := t.pending[hash]; tx != nil {
		return TxStatusPending
	}

	if tx := t.queue[hash]; tx != nil {
		return TxStatusQueued
	}

	return TxStatusUnknown
}


// Add adds a transaction to the lookup.
func (t *txLookup) Add(tx *types.Transaction, pending bool) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// t.slots += numSlots(tx)
	// slotsGauge.Update(int64(t.slots))

	if pending {
		t.pending[tx.Hash()] = tx
	} else {
		t.queue[tx.Hash()] = tx
	}
}


// Remove removes a transaction from the lookup.
func (t *txLookup) Remove(hash common.Hash) {
	t.lock.Lock()
	defer t.lock.Unlock()

	tx, ok := t.pending[hash]
	if !ok {
		tx, ok = t.queue[hash]
	}
	if !ok {
		log.Error("No transaction found to be deleted", "hash", hash)
		return
	}
	// t.slots -= numSlots(tx)
	// slotsGauge.Update(int64(t.slots))

	delete(t.pending, hash)
	delete(t.queue, hash)
}

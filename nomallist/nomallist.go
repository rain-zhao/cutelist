package nomallist

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// IntList an struct of list order asc implement by linked list
// note: list would not contain same value
type IntList struct {
	head   *intNode
	length int64
}

type intNode struct {
	value  int
	next   *intNode
	marked uint32
	mu     sync.Mutex
}

const (
	UNMARKED = iota
	MARKED
)

func newIntNode(value int) *intNode {
	return &intNode{value: value}
}

// loadNext load node's next node atomic
func (n *intNode) loadNext() *intNode {
	return (*intNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&n.next))))
}

// storeNext store node's next node atomic
func (n *intNode) storeNext(node *intNode) {
	//same as n.next = node
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&n.next)), unsafe.Pointer(node))
}

// isMarked check node's marked
func (n *intNode) isMarked() bool {
	return atomic.LoadUint32(&n.marked) == 1
}

// setMarked set node's marked flag as flag
func (n *intNode) setMarked(flag uint32) {
	atomic.StoreUint32(&n.marked, flag)
}

// NewInt return one new list
func NewInt() *IntList {
	return &IntList{head: newIntNode(0)}
}

// Contains check whether value is in l
func (l *IntList) Contains(value int) bool {

	cur := l.head

	// skip dummy head
	cur = cur.loadNext()

	// find first appear of val
	for cur != nil && cur.value < value {
		cur = cur.loadNext()
	}

	return cur != nil && cur.value == value && !cur.isMarked()
}

// Insert insert node of value into orderd list
func (l *IntList) Insert(value int) bool {

	var prev, cur *intNode

	for {
		// 1: find the pos
		pprev, ccur, exist := l.find(value)

		//value already exist
		if exist {
			return false
		}

		// 2.1 lock the prev node
		pprev.mu.Lock()
		// 2.2 check prev.next == cur and !prev.marked.
		//no need to use nextNode() cause prev.next has been locked
		if pprev.next != ccur || pprev.isMarked() {
			pprev.mu.Unlock()
			continue
		}

		prev, cur = pprev, ccur
		break
	}

	// 3 new node
	newNode := newIntNode(value)

	// 4 insert into list
	newNode.next = cur
	prev.storeNext(newNode)

	// 5 unlock prev
	prev.mu.Unlock()

	// increase list's len
	atomic.AddInt64(&l.length, 1)

	return true
}

func (l *IntList) Delete(value int) bool {

	var prev, cur *intNode

	for {
		// 1: find the pos
		pprev, ccur, exist := l.find(value)

		//value not exist
		if !exist {
			return false
		}

		// 2 lock cur and check cur.marked
		ccur.mu.Lock()
		if ccur.isMarked() {
			ccur.mu.Unlock()
			continue
		}

		// 3 lock prev and ckeck prev.next == cur and !prev.mark
		pprev.mu.Lock()
		if pprev.next != ccur || pprev.isMarked() {
			//unlock prev first
			pprev.mu.Unlock()
			ccur.mu.Unlock()
			continue
		}
		prev, cur = pprev, ccur
		break
	}

	// 4 delete node from list
	cur.setMarked(MARKED)
	prev.storeNext(cur.next)

	// unlock prev and cur
	prev.mu.Unlock()
	cur.mu.Unlock()

	// desc list's len
	atomic.AddInt64(&l.length, -1)

	return true
}

// Range iterate each node of l and put node.value as f's input param
// if f return false then stop iteration
func (l *IntList) Range(f func(value int) bool) {
	cur := l.head

	// skip dummy head
	cur = cur.loadNext()

	// iterate
	for cur != nil {
		if cur.isMarked() {
			cur = cur.loadNext()
			continue
		}
		if !f(cur.value) {
			break
		}
		cur = cur.loadNext()
	}
}

// Len return length of l
func (l *IntList) Len() int {
	return int(atomic.LoadInt64(&l.length))
}

// find find the node whose val == value and its pre node
func (l *IntList) find(value int) (prev, cur *intNode, exist bool) {
	prev, cur = l.head, l.head.loadNext()

	for cur != nil && cur.value < value {

		prev = cur
		cur = cur.loadNext()
	}

	if cur != nil && cur.value == value {
		exist = true
	}
	return
}

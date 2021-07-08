package nomallist

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

//IntList an struct of list order asc implement by linked list
//note: list would not contain same value
type IntList struct {
	head   *intNode
	length int64
}

type intNode struct {
	value  int
	next   *intNode
	marked bool
	mu     sync.Mutex
}

func newIntNode(value int) *intNode {
	return &intNode{value: value}
}

//nextNode load node's next node
func (n *intNode) nextNode() *intNode {
	return (*intNode)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&n.next))))
}

// return one new list
func NewInt() *IntList {
	return &IntList{head: newIntNode(0)}
}

//Contains check whether value is in l
func (l *IntList) Contains(value int) bool {

	cur := l.head

	//skip dummy head
	cur = cur.nextNode()

	//find first appear of val
	for cur != nil && cur.value < value {
		cur = cur.nextNode()
	}

	return cur != nil && cur.value == value && !cur.marked
}

//Insert insert node of value into orderd list
func (l *IntList) Insert(value int) bool {

	prev, cur := l.head, l.head.nextNode()

	for {
		//1: find the first node's val > value and its pre node
		if cur != nil && cur.value < value {
			prev = cur
			cur = cur.nextNode()
		}

		//value already exist
		if cur != nil && cur.value == value {
			return false
		}

		//2.1 lock the prev node
		prev.mu.Lock()
		//2.2 check prev.next == cur and !prev.marked.
		//no need to use nextNode() cause prev.next has been locked
		if prev.next != cur || prev.marked {
			prev.mu.Unlock()
			continue
		}

		break
	}

	//3 new node
	newNode := newIntNode(value)

	//4 insert into list
	prev.next, newNode.next = newNode, cur

	//5 unlock prev
	prev.mu.Unlock()

	//increase list's len
	atomic.AddInt64(&l.length, 1)

	return true
}

func (l *IntList) Delete(value int) bool {

	prev, cur := l.head, l.head.nextNode()

	for {
		//1: find the node's val == value and its pre node
		if cur != nil && cur.value < value {
			prev = cur
			cur = cur.nextNode()
		}

		//value not exist
		if cur == nil || cur.value != value {
			return false
		}

		//2 lock cur and check cur.marked
		cur.mu.Lock()
		if cur.marked {
			cur.mu.Unlock()
			continue
		}

		//3 lock prev and ckeck prev.next == cur and !prev.mark
		prev.mu.Lock()
		if prev.next != cur || prev.marked {
			//unlock prev first
			prev.mu.Unlock()
			cur.mu.Unlock()
			continue
		}
		break
	}

	//4 delete node from list
	cur.marked = true
	prev.next = cur.next

	//unlock prev and cur
	prev.mu.Unlock()
	cur.mu.Unlock()

	//desc list's len
	atomic.AddInt64(&l.length, -1)

	return true
}

//Range iterate each node of l and put node.value as f's input param
// if f return false then stop iteration
func (l *IntList) Range(f func(value int) bool) {
	cur := l.head

	//skip dummy head
	cur = cur.nextNode()

	//iterate
	//TODO  whether cur.marked would continue the iteration?
	for cur != nil && !cur.marked && f(cur.value) {
		cur = cur.nextNode()
	}
}

//Len return length of l
func (l *IntList) Len() int {
	return int(atomic.LoadInt64(&l.length))
}

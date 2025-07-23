package btree

import (
	"cmp"
	"iter"
)

type LLBTree[K, V any] struct {
	cmp  func(l, r K) int
	root *llbtreeNode[K, V]
}

func NewLLBTree[K, V any](cmp func(l, r K) int) *LLBTree[K, V] {
	return &LLBTree[K, V]{
		cmp: cmp,
	}
}

func NewLLBTreeOrdered[K cmp.Ordered, V any]() *LLBTree[K, V] {
	return &LLBTree[K, V]{
		cmp: cmp.Compare[K],
	}
}

func (t *LLBTree[K, V]) setRoot(x *llbtreeNode[K, V]) {
	t.root = x
	if x != nil {
		x.parent = nil
	}
}

func (t *LLBTree[K, V]) replaceChild(p, old, x *llbtreeNode[K, V]) {
	switch {
	case p == nil:
		if t.root != old {
			panic("corrupt llrb")
		}
		t.setRoot(x)
	case p.left == old:
		p.setLeft(x)
	case p.right == old:
		p.setRight(x)
	default:
		panic("corrupt llrb")
	}
}

func (t *LLBTree[K, V]) rotateRight(y *llbtreeNode[K, V]) (newRoot *llbtreeNode[K, V]) {
	// turning (y (x a b) c) into (x a (y b c)).
	//     y
	//    / \
	//   x   c
	//  / \
	// a   b
	//
	// ↓
	//
	//    x
	//   / \
	//  a   y
	//     / \
	//    b   c
	p := y.parent
	x := y.left
	b := x.right

	x.setRight(y)
	y.setLeft(b)
	t.replaceChild(p, y, x)

	x.red = y.red
	y.red = true

	return x
}

func (t *LLBTree[K, V]) rotateLeft(x *llbtreeNode[K, V]) (newRoot *llbtreeNode[K, V]) {
	//   x
	//  / \
	// a   y
	//    / \
	//   b   c
	//
	// ↓
	//
	//     y
	//    / \
	//   x   c
	//  / \
	// a   b
	p := x.parent
	y := x.right
	b := y.left

	y.setLeft(x)
	x.setRight(b)
	t.replaceChild(p, x, y)

	y.red = x.red
	x.red = true

	return y
}

// insertBtree does standard btree insertion.
func (t *LLBTree[K, V]) insertBtree(k K, v V) *llbtreeNode[K, V] {
	if t.root == nil {
		t.root = &llbtreeNode[K, V]{
			cmp:   t.cmp,
			red:   true,
			key:   k,
			value: v,
		}
		return t.root
	}
	loc, parent := t.root.findLocation(&t.root, k)
	if *loc != nil {
		(*loc).value = v
	} else {
		*loc = &llbtreeNode[K, V]{
			cmp:    t.cmp,
			parent: parent,
			red:    true,
			key:    k,
			value:  v,
		}
	}
	return *loc
}

func (t *LLBTree[K, V]) fixup(tgt *llbtreeNode[K, V]) {
	// 1.	Every node is either red or black.
	// 2. A NIL node is considered black.
	// 3. A red node does not have a red child.
	// 4. Every path from a given node to any of its descendant NIL nodes goes through the same number of black nodes.
	// 5. The root is black (by convention).
	// 6. If a node has only one red child, it must be the left child.
	n := tgt
	for n != nil {
		if n.right.isRed() && n.left.isBlack() {
			// 6. If a node has only one red child, it must be the left child.

			//      x
			//    /   \
			//   a   (y)
			//       / \
			//      b   c
			//
			// ↓
			//
			//     y (y may be red if x was red)
			//    /  \
			//  (x)   c
			//  / \
			// a  b
			n = t.rotateLeft(n)
			// n is y
		}
		if n.left.isRed() && n.left.left.isRed() {
			// 3. A red node does not have a red child.

			//
			//      y
			//    /   \
			//  (x)    c
			//  /  \
			// (a)  b
			//
			// ↓
			//
			//      x (x may be red if y was red)
			//    /   \
			//  (a)   (y)
			//        / \
			//       b   c
			n = t.rotateRight(n)
			// flipColor is called right after this
		}
		if n.left.isRed() && n.right.isRed() {
			n.flipColor()
		}
		n = n.parent
	}
	if t.root != nil {
		t.root.red = false
	}
}

func (t *LLBTree[K, V]) insert(k K, v V) {
	node := t.insertBtree(k, v)
	t.fixup(node)
}

func (t *LLBTree[K, V]) moveRedRight(x *llbtreeNode[K, V]) *llbtreeNode[K, V] {
	x.flipColor()
	if x.left != nil && x.left.left != nil && x.left.left.red {
		x = t.rotateRight(x)
		x.flipColor()
	}
	return x
}

func (t *LLBTree[K, V]) moveRedLeft(x *llbtreeNode[K, V]) *llbtreeNode[K, V] {
	x.flipColor()
	if x.right != nil && x.right.left.isRed() {
		t.rotateRight(x.right)
		x = t.rotateLeft(x)
		x.flipColor()
	}
	return x
}

func (t *LLBTree[K, V]) deleteMin(zpos **llbtreeNode[K, V]) (z, zparent *llbtreeNode[K, V]) {
	z = *zpos
	for {
		if z.left == nil {
			zparent = z.parent
			if z.right != nil {
				panic("bad z.right")
			}
			*zpos = nil
			return z, zparent
		}
		if !z.left.isRed() && !z.left.left.isRed() {
			z = t.moveRedLeft(z)
		}
		zpos, z = &z.left, z.left
	}
}

func (t *LLBTree[K, V]) delete(key K) (deleted bool) {
	pos, parent, x := &t.root, t.root, t.root
	if x == nil {
		return false
	}
	for {
		if x == nil {
			t.fixup(parent)
			return
		}
		if t.cmp(key, x.key) < 0 {
			if !x.left.isRed() && x.left != nil && !x.left.left.isRed() {
				x = t.moveRedLeft(x)
			}
			parent, pos, x = x, &x.left, x.left
		} else {
			if x.left.isRed() {
				x = t.rotateRight(x)
			}
			if t.cmp(key, x.key) == 0 && x.right == nil {
				*pos = nil
				t.fixup(parent)
				break
			}
			if !x.right.isRed() && x.right != nil && !x.right.left.isRed() {
				x = t.moveRedRight(x)
			}
			if t.cmp(key, x.key) == 0 {
				z, zparent := t.deleteMin(&x.right)
				if zparent == x {
					zparent = z
				}
				z.setLeft(x.left)
				z.setRight(x.right)
				z.red = x.red
				t.replaceChild(x.parent, x, z)
				t.fixup(zparent)
				break
			}
			parent, pos, x = x, &x.right, x.right
		}
	}

	x.parent = nil
	x.left = nil
	x.right = nil
	x.deleted = true

	return true
}

func (t *LLBTree[K, V]) get(k K) *llbtreeNode[K, V] {
	if t.root == nil {
		return nil
	}
	loc, _ := t.root.findLocation(&t.root, k)
	if loc == nil {
		return nil
	}
	return *loc
}

func (t *LLBTree[K, V]) all() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if t.root == nil {
			return
		}
		n := t.root.leftMost(&t.root)
		if n == nil || *n == nil {
			return
		}
		current := *n
		if !yield(current.key, current.value) {
			return
		}
		var next *llbtreeNode[K, V]
		for {
			if current == nil {
				return
			}
			if current.deleted {
				next = t.root.nextAfter(&t.root, current.key)
			} else {
				next = current.next(&t.root)
			}
			if next == nil {
				return
			}
			if !yield(next.key, next.value) {
				return
			}
			current = next
		}
	}
}

type llbtreeNode[K, V any] struct {
	cmp                 func(l, r K) int
	parent, left, right *llbtreeNode[K, V]
	deleted             bool
	red                 bool
	key                 K
	value               V
}

func (n *llbtreeNode[K, V]) loc(root **llbtreeNode[K, V]) **llbtreeNode[K, V] {
	if n.parent == nil {
		return root
	}
	if n.parent.left == n {
		return &n.parent.left
	} else {
		return &n.parent.right
	}
}

func (n *llbtreeNode[K, V]) findLocation(root **llbtreeNode[K, V], k K) (loc **llbtreeNode[K, V], parent *llbtreeNode[K, V]) {
	if n == nil {
		return nil, nil
	}
	loc = n.loc(root)
	for *loc != nil {
		switch n.cmp(k, (*loc).key) {
		case 0:
			return
		case 1: // k > node.key
			loc, parent = &(*loc).right, *loc
		default: // k < node.key
			loc, parent = &(*loc).left, *loc
		}
	}
	return
}

func (n *llbtreeNode[K, V]) next(root **llbtreeNode[K, V]) *llbtreeNode[K, V] {
	if n == nil {
		return nil
	}
	if n.isLeaf() {
		if n.parent == nil {
			// root
			return nil
		}
		return n.ascendFromRight()
	}
	if n.right != nil {
		node := n.right.leftMost(root)
		if node == nil {
			return nil
		}
		return *node
	}
	return n.ascendFromRight()
}

func (n *llbtreeNode[K, V]) nextAfter(root **llbtreeNode[K, V], k K) *llbtreeNode[K, V] {
	if n == nil {
		return nil
	}
	loc, parent := n.findLocation(root, k)
	if *loc == nil {
		if loc == &parent.left {
			return parent
		} else {
			return parent.ascendFromRight()
		}
	}
	return (*loc).next(root)
}

func (n *llbtreeNode[K, V]) ascendFromRight() *llbtreeNode[K, V] {
	if n == nil {
		return nil
	}
	node := n
	for node.parent != nil {
		if node.parent.left == node {
			return node.parent
		}
		node = node.parent
	}
	return nil
}

func (n *llbtreeNode[K, V]) leftMost(root **llbtreeNode[K, V]) **llbtreeNode[K, V] {
	if n == nil {
		return nil
	}
	node := n.loc(root)
	for (*node).left != nil {
		node = &(*node).left
	}
	return node
}

func (n *llbtreeNode[K, V]) isLeaf() bool {
	if n == nil {
		return true
	}
	return n.left == nil && n.right == nil
}

func (n *llbtreeNode[K, V]) isRed() bool {
	if n == nil {
		return false
	}
	return n.red
}

func (n *llbtreeNode[K, V]) isBlack() bool {
	return !n.isRed()
}

func (n *llbtreeNode[K, V]) unlink() {
	if n == nil {
		return
	}
	n.deleted = true
	if n.parent != nil {
		if n.parent.left == n {
			n.parent.left = nil
		} else {
			n.parent.right = nil
		}
	}
	n.parent = nil
	n.left = nil
	n.right = nil
}

func (n *llbtreeNode[K, V]) flipColor() {
	n.red = !n.red
	if n.left != nil {
		n.left.red = !n.left.red
	}
	if n.right != nil {
		n.right.red = !n.right.red
	}
}

func (n *llbtreeNode[K, V]) updateKeyValue(updater *llbtreeNode[K, V]) {
	n.red = updater.red
	n.key = updater.key
	n.value = updater.value
}

func (n *llbtreeNode[K, V]) updateParent(updater *llbtreeNode[K, V]) {
	n.parent = updater.parent
	if updater.parent != nil {
		if updater.parent.left == updater {
			updater.parent.left = n
		} else {
			updater.parent.right = n
		}
	}
}

func (n *llbtreeNode[K, V]) updateChildren(updater *llbtreeNode[K, V]) {
	n.left = updater.left
	n.right = updater.right
}

func (n *llbtreeNode[K, V]) setRight(node *llbtreeNode[K, V]) {
	n.right = node
	if node != nil {
		node.parent = n
	}
}

func (n *llbtreeNode[K, V]) setLeft(node *llbtreeNode[K, V]) {
	n.left = node
	if node != nil {
		node.parent = n
	}
}

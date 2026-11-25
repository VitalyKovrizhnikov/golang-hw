package hw04lrucache

type List interface {
	Len() int
	Front() *ListItem
	Back() *ListItem
	PushFront(v interface{}) *ListItem
	PushBack(v interface{}) *ListItem
	Remove(i *ListItem)
	MoveToFront(i *ListItem)
}

type ListItem struct {
	Value interface{}
	Next  *ListItem
	Prev  *ListItem
}

type list struct {
	front  *ListItem
	back   *ListItem
	length int
}

func NewList() List {
	return new(list)
}

func (l *list) Len() int {
	return l.length
}

func (l *list) Front() *ListItem {
	return l.front
}

func (l *list) Back() *ListItem {
	return l.back
}

func (l *list) PushFront(v interface{}) *ListItem {
	item := &ListItem{Value: v}
	l.pushFront(item)

	return item
}

func (l *list) PushBack(v interface{}) *ListItem {
	item := &ListItem{Value: v}
	l.pushBack(item)

	return item
}

func (l *list) Remove(i *ListItem) {
	l.unlink(i)
}

func (l *list) MoveToFront(i *ListItem) {
	if i == l.front {
		return
	}

	l.unlink(i)
	l.pushFront(i)
}

func (l *list) pushFront(item *ListItem) {
	item.Prev = nil
	item.Next = l.front

	if l.front != nil {
		l.front.Prev = item
	} else {
		l.back = item
	}

	l.front = item
	l.length++
}

func (l *list) pushBack(item *ListItem) {
	item.Next = nil
	item.Prev = l.back

	if l.back != nil {
		l.back.Next = item
	} else {
		l.front = item
	}

	l.back = item
	l.length++
}

func (l *list) unlink(item *ListItem) {
	if item.Prev != nil {
		item.Prev.Next = item.Next
	} else {
		l.front = item.Next
	}

	if item.Next != nil {
		item.Next.Prev = item.Prev
	} else {
		l.back = item.Prev
	}

	item.Prev = nil
	item.Next = nil
	l.length--
}

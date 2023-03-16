package xactor

import "context"

type result struct {
	resp interface{}
	err  error
}

type mail struct {
	ctx      context.Context
	req      interface{}
	t        mailType
	resultCh chan *result
}

func newMail(ctx context.Context, t mailType, req interface{}) *mail {
	return &mail{ctx: ctx, t: t, req: req, resultCh: make(chan *result, 1)}
}

type mailBox struct {
	mailCh chan *mail
}

func newMailBox() *mailBox {
	box := &mailBox{
		mailCh: make(chan *mail, mailMaxCount),
	}
	return box
}

func (box *mailBox) recvMail() <-chan *mail {
	return box.mailCh
}

func (box *mailBox) sendMail(m *mail) {
	box.mailCh <- m
}

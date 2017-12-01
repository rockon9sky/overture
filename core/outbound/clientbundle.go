// Copyright (c) 2016 shawn1m. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package outbound

import (
	"github.com/miekg/dns"
	"github.com/rockon9sky/overture/core/cache"
	"github.com/rockon9sky/overture/core/common"
	"github.com/rockon9sky/overture/core/hosts"
)

type ClientBundle struct {
	ResponseMessage *dns.Msg
	QuestionMessage *dns.Msg

	ClientList []*Client

	DNSUpstreamList []*DNSUpstream
	InboundIP       string

	Hosts *hosts.Hosts
	Cache *cache.Cache
}

func NewClientBundle(q *dns.Msg, ul []*DNSUpstream, ip string, h *hosts.Hosts, cache *cache.Cache) *ClientBundle {

	cb := &ClientBundle{QuestionMessage: q, DNSUpstreamList: ul, InboundIP: ip, Hosts: h, Cache: cache}

	for _, u := range ul {

		c := NewClient(cb.QuestionMessage, u, cb.InboundIP, cb.Hosts, cb.Cache)
		cb.ClientList = append(cb.ClientList, c)
	}

	return cb
}

func (cb *ClientBundle) ExchangeFromRemote(isCache bool, isLog bool) {

	ch := make(chan *dns.Msg, len(cb.ClientList))

	for _, o := range cb.ClientList {
		go func(c *Client, ch chan *dns.Msg) {
			c.ExchangeFromRemote(isCache, isLog)
			ch <- c.ResponseMessage
		}(o, ch)
	}

	var em *dns.Msg

	for i := 0; i < len(cb.ClientList); i++ {
		if m := <-ch; m != nil {
			if common.IsAnswerEmpty(m) {
				em = m
				break
			}
			cb.ResponseMessage = m
			return
		}
	}
	cb.ResponseMessage = em
}

func (cb *ClientBundle) ExchangeFromLocal() bool {

	for _, c := range cb.ClientList {
		if c.ExchangeFromLocal() {
			cb.ResponseMessage = c.ResponseMessage
			c.logAnswer(true)
			return true
		}
	}
	return false
}

func (cb *ClientBundle) UpdateFromDNSUpstream(ul []*DNSUpstream) {

	cb.DNSUpstreamList = ul
	cb.ResponseMessage = nil

	var cl []*Client

	for _, u := range ul {
		c := NewClient(cb.QuestionMessage, u, cb.InboundIP, cb.Hosts, cb.Cache)
		cl = append(cl, c)
	}

	cb.ClientList = cl
}

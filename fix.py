# -*- coding: utf-8 -*-
import re

with open('internal/proxy/session.go', 'r', encoding='utf-8') as f:
    content = f.read()

ring_buffer_code = '''
type HashrateRingBuffer struct {
	mu           sync.RWMutex
	MainHash     [360]float64
	FeeHash      [360]float64
	CurrentIndex int
}

func (r *HashrateRingBuffer) AddShare(diff float64, isFee bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if isFee {
		r.FeeHash[r.CurrentIndex] += diff
	} else {
		r.MainHash[r.CurrentIndex] += diff
	}
}

func (r *HashrateRingBuffer) Tick() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.CurrentIndex = (r.CurrentIndex + 1) % 360
	r.MainHash[r.CurrentIndex] = 0
	r.FeeHash[r.CurrentIndex] = 0
}
'''

if 'HashrateRingBuffer' not in content:
    content += '\n' + ring_buffer_code

if 'RingBuffer *HashrateRingBuffer' not in content:
    content = content.replace('CurrentFeeMode FeeMode\n', 'CurrentFeeMode FeeMode\n\tRingBuffer *HashrateRingBuffer\n')

if '&HashrateRingBuffer{}' not in content:
    content = content.replace('CurrentFeeMode: FeeModeNone,\n', 'CurrentFeeMode: FeeModeNone,\n\t\tRingBuffer: &HashrateRingBuffer{},\n')

content = content.replace('if method == \"mining.notify\" || method == \"eth_getWork\" {', 'if method == \"mining.notify\" || method == \"eth_getWork\" || method == \"mining.set_extranonce\" {')

intercept_code = '''						// 拦截抽水池下发的 extranonce，绝对不将其转发给矿机
						continue'''
content = content.replace(intercept_code, '// continue // Do NOT intercept extranonce')

stop_fee = '''		if !s.InBandFeeActive {
			if s.Config.EnableAsic && s.Protocol != \"ETH_PROXY\" {
				if !isExploit && (s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size || s.FeeExtranonce.En1 != s.MainExtranonce.En1) {
					extranonceToSend = s.MainExtranonce'''
new_stop_fee = '''		if !s.InBandFeeActive {
			if s.Config.EnableAsic && s.Protocol != \"ETH_PROXY\" {
				if s.FeeExtranonce == nil || s.MainExtranonce == nil || s.FeeExtranonce.En2Size != s.MainExtranonce.En2Size || s.FeeExtranonce.En1 != s.MainExtranonce.En1 {
					extranonceToSend = s.MainExtranonce'''
content = content.replace(stop_fee, new_stop_fee)

content = content.replace('\tisExploit := s.IsF2PoolExploit\n', '')

stop_fee_diff = '''			localDiff := s.LocalDiff
			if localDiff > 0 {
				difficultyToSend = localDiff
			}'''
new_stop_fee_diff = '''			localDiff := s.LocalDiff
			if localDiff > 0 && localDiff != s.CurrentDiff {
				difficultyToSend = localDiff
			}'''
content = content.replace(stop_fee_diff, new_stop_fee_diff)

with open('internal/proxy/session.go', 'w', encoding='utf-8') as f:
    f.write(content)

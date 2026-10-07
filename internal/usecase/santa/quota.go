package santa

import (
	"sync"
	"time"
)

const (
	// emailCodesPerHour — сколько кодов можно отправить на один адрес за
	// emailQuotaWindow из любых комнат. Кулдаун кода — на участника, а вступить
	// в открытую комнату может кто угодно и сколько угодно раз.
	emailCodesPerHour = 5
	emailQuotaWindow  = time.Hour
)

// emailQuota — скользящее окно отправок по адресу в памяти процесса. На
// нескольких экземплярах предел умножается на их число — для защиты от
// рассылки этого достаточно, базе лишняя таблица не нужна.
type emailQuota struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	sent   map[string][]time.Time
	swept  time.Time
}

func newEmailQuota(limit int, window time.Duration) *emailQuota {
	return &emailQuota{limit: limit, window: window, sent: map[string][]time.Time{}}
}

// take занимает отправку на адрес; false — предел за окно исчерпан.
func (q *emailQuota) take(email string, now time.Time) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	since := now.Add(-q.window)
	if now.Sub(q.swept) > q.window {
		for addr, times := range q.sent {
			if len(recent(times, since)) == 0 {
				delete(q.sent, addr)
			}
		}
		q.swept = now
	}
	times := recent(q.sent[email], since)
	if len(times) >= q.limit {
		q.sent[email] = times
		return false
	}
	q.sent[email] = append(times, now)
	return true
}

// recent — отметки позже since; times упорядочены по возрастанию.
func recent(times []time.Time, since time.Time) []time.Time {
	i := 0
	for i < len(times) && !times[i].After(since) {
		i++
	}
	return times[i:]
}

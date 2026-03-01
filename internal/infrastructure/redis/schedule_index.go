package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"greenpause/internal/domain"
)

type ScheduleIndex struct {
	addr         string
	keyPrefix    string
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func NewScheduleIndex(addr string) *ScheduleIndex {
	return &ScheduleIndex{
		addr:         addr,
		keyPrefix:    "ScheduleIndex",
		dialTimeout:  2 * time.Second,
		readTimeout:  2 * time.Second,
		writeTimeout: 2 * time.Second,
	}
}

func (s *ScheduleIndex) Upsert(ctx context.Context, tenantID domain.TenantID, dueAtUtc time.Time, reminderID domain.ReminderID) error {
	key := s.tenantKey(tenantID)
	score := strconv.FormatInt(dueAtUtc.UTC().UnixMilli(), 10)
	_, err := s.exec(ctx, "ZADD", key, score, string(reminderID))
	return err
}

func (s *ScheduleIndex) Remove(ctx context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) error {
	key := s.tenantKey(tenantID)
	_, err := s.exec(ctx, "ZREM", key, string(reminderID))
	return err
}

func (s *ScheduleIndex) Score(ctx context.Context, tenantID domain.TenantID, reminderID domain.ReminderID) (*float64, error) {
	key := s.tenantKey(tenantID)
	resp, err := s.exec(ctx, "ZSCORE", key, string(reminderID))
	if err != nil {
		return nil, err
	}
	if resp.kind == redisRespNullBulk {
		return nil, nil
	}

	value, err := strconv.ParseFloat(resp.stringValue(), 64)
	if err != nil {
		return nil, fmt.Errorf("parse zscore: %w", err)
	}
	return &value, nil
}

func (s *ScheduleIndex) tenantKey(tenantID domain.TenantID) string {
	return s.keyPrefix + ":" + string(tenantID)
}

func (s *ScheduleIndex) exec(ctx context.Context, args ...string) (redisResp, error) {
	if strings.TrimSpace(s.addr) == "" {
		return redisResp{}, errors.New("redis address is required")
	}

	dialer := net.Dialer{Timeout: s.dialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", s.addr)
	if err != nil {
		return redisResp{}, err
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
		_ = conn.SetReadDeadline(time.Now().Add(s.readTimeout))
	}

	if _, err := conn.Write(encodeCommand(args...)); err != nil {
		return redisResp{}, err
	}

	resp, err := readResp(bufio.NewReader(conn))
	if err != nil {
		return redisResp{}, err
	}
	if resp.kind == redisRespError {
		return redisResp{}, errors.New(resp.value)
	}
	return resp, nil
}

func encodeCommand(args ...string) []byte {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

type redisRespKind int

const (
	redisRespSimple redisRespKind = iota + 1
	redisRespError
	redisRespInteger
	redisRespBulk
	redisRespNullBulk
)

type redisResp struct {
	kind  redisRespKind
	value string
	i64   int64
}

func (r redisResp) stringValue() string {
	switch r.kind {
	case redisRespInteger:
		return strconv.FormatInt(r.i64, 10)
	default:
		return r.value
	}
}

func readResp(r *bufio.Reader) (redisResp, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return redisResp{}, err
	}

	line, err := r.ReadString('\n')
	if err != nil {
		return redisResp{}, err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")

	switch prefix {
	case '+':
		return redisResp{kind: redisRespSimple, value: line}, nil
	case '-':
		return redisResp{kind: redisRespError, value: line}, nil
	case ':':
		i, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return redisResp{}, err
		}
		return redisResp{kind: redisRespInteger, i64: i}, nil
	case '$':
		lenN, err := strconv.Atoi(line)
		if err != nil {
			return redisResp{}, err
		}
		if lenN == -1 {
			return redisResp{kind: redisRespNullBulk}, nil
		}
		buf := make([]byte, lenN+2)
		if _, err := readFull(r, buf); err != nil {
			return redisResp{}, err
		}
		return redisResp{kind: redisRespBulk, value: string(buf[:lenN])}, nil
	default:
		return redisResp{}, fmt.Errorf("unsupported redis response prefix: %q", prefix)
	}
}

func readFull(r *bufio.Reader, dst []byte) (int, error) {
	total := 0
	for total < len(dst) {
		n, err := r.Read(dst[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

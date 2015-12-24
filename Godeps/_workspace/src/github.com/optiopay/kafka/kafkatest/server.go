package kafkatest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/optiopay/kafka/proto"
)

type Server struct {
	mu          sync.RWMutex
	brokers     []proto.MetadataRespBroker
	topics      map[string]map[int32][]*proto.Message
	ln          net.Listener
	middlewares []Middleware
}

// Middleware is function that is called for every incomming kafka message,
// before running default processing handler. Middleware function can return
// nil or kafka response message.
type Middleware func(nodeID int32, requestKind int16, content []byte) Response

// Response is any kafka response as defined in kafka/proto package
type Response interface {
	Bytes() ([]byte, error)
}

// NewServer return new mock server instance. Any number of middlewares can be
// passed to customize request handling. For every incomming request, all
// middlewares are called one after another in order they were passed. If any
// middleware return non nil response message, response is instasntly written
// to the client and no further code execution for the request is made -- no
// other middleware is called nor the default handler is executed.
func NewServer(middlewares ...Middleware) *Server {
	s := &Server{
		brokers:     make([]proto.MetadataRespBroker, 0),
		topics:      make(map[string]map[int32][]*proto.Message),
		middlewares: middlewares,
	}
	return s
}

// Addr return server instance address or empty string if not running.
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ln != nil {
		return s.ln.Addr().String()
	}
	return ""
}

// Reset will clear out local messages and topics.
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.topics = make(map[string]map[int32][]*proto.Message)
}

// Close shut down server if running. It is safe to call it more than once.
func (s *Server) Close() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln != nil {
		err = s.ln.Close()
		s.ln = nil
	}
	return err
}

// ServeHTTP provides JSON serialized server state information.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	topics := make(map[string]map[string][]*proto.Message)
	for name, parts := range s.topics {
		topics[name] = make(map[string][]*proto.Message)
		for part, messages := range parts {
			topics[name][strconv.Itoa(int(part))] = messages
		}
	}

	w.Header().Set("content-type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"topics":  topics,
		"brokers": s.brokers,
	})
	if err != nil {
		log.Printf("cannot JSON encode state: %s", err)
	}
}

// AddMessages append messages to given topic/partition. If topic or partition
// does not exists, it is being created.
// To only create topic/partition, call this method withough giving any
// message.
func (s *Server) AddMessages(topic string, partition int32, messages ...*proto.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parts, ok := s.topics[topic]
	if !ok {
		parts = make(map[int32][]*proto.Message)
		s.topics[topic] = parts
	}

	for i := int32(0); i <= partition; i++ {
		if _, ok := parts[i]; !ok {
			parts[i] = make([]*proto.Message, 0)
		}
	}
	if len(messages) > 0 {
		start := len(parts[partition])
		for i, msg := range messages {
			msg.Offset = int64(start + i)
			msg.Partition = partition
			msg.Topic = topic
		}
		parts[partition] = append(parts[partition], messages...)
	}
}

// Run starts kafka mock server listening on given address.
func (s *Server) Run(addr string) error {
	const nodeID = 100

	s.mu.RLock()
	if s.ln != nil {
		s.mu.RUnlock()
		log.Printf("server already running: %s", s.ln.Addr())
		return fmt.Errorf("server already running: %s", s.ln.Addr())
	}

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		s.mu.RUnlock()
		log.Printf("cannot listen on address %q: %s", addr, err)
		return fmt.Errorf("cannot listen: %s", err)
	}
	defer func() {
		_ = ln.Close()
	}()
	s.ln = ln

	if host, port, err := net.SplitHostPort(ln.Addr().String()); err != nil {
		s.mu.RUnlock()
		log.Printf("cannot extract host/port from %q: %s", ln.Addr(), err)
		return fmt.Errorf("cannot extract host/port from %q: %s", ln.Addr(), err)
	} else {
		prt, err := strconv.Atoi(port)
		if err != nil {
			s.mu.RUnlock()
			log.Printf("invalid port %q: %s", port, err)
			return fmt.Errorf("invalid port %q: %s", port, err)
		}
		s.brokers = append(s.brokers, proto.MetadataRespBroker{
			NodeID: nodeID,
			Host:   host,
			Port:   int32(prt),
		})
	}
	s.mu.RUnlock()

	for {
		conn, err := ln.Accept()
		if err == nil {
			go s.handleClient(nodeID, conn)
		}
	}
}

// MustSpawn run server in the background on random port. It panics if server
// cannot be spawned.
// Use Close method to stop spawned server.
func (s *Server) MustSpawn() {
	const nodeID = 100

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ln != nil {
		return
	}

	ln, err := net.Listen("tcp4", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("cannot listen: %s", err))
	}
	s.ln = ln

	if host, port, err := net.SplitHostPort(ln.Addr().String()); err != nil {
		panic(fmt.Sprintf("cannot extract host/port from %q: %s", ln.Addr(), err))
	} else {
		prt, err := strconv.Atoi(port)
		if err != nil {
			panic(fmt.Sprintf("invalid port %q: %s", port, err))
		}
		s.brokers = append(s.brokers, proto.MetadataRespBroker{
			NodeID: nodeID,
			Host:   host,
			Port:   int32(prt),
		})
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err == nil {
				go s.handleClient(nodeID, conn)
			}
		}
	}()
}

func (s *Server) handleClient(nodeID int32, conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	for {
		kind, b, err := proto.ReadReq(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("client read error: %s", err)
			}
			return
		}

		var resp response

		for _, middleware := range s.middlewares {
			resp = middleware(nodeID, kind, b)
			if resp != nil {
				break
			}
		}

		if resp == nil {
			switch kind {
			case proto.ProduceReqKind:
				req, err := proto.ReadProduceReq(bytes.NewBuffer(b))
				if err != nil {
					log.Printf("cannot parse produce request: %s\n%s", err, b)
					return
				}
				resp = s.handleProduceRequest(nodeID, conn, req)
			case proto.FetchReqKind:
				req, err := proto.ReadFetchReq(bytes.NewBuffer(b))
				if err != nil {
					log.Printf("cannot parse fetch request: %s\n%s", err, b)
					return
				}
				resp = s.handleFetchRequest(nodeID, conn, req)
			case proto.OffsetReqKind:
				req, err := proto.ReadOffsetReq(bytes.NewBuffer(b))
				if err != nil {
					log.Printf("cannot parse offset request: %s\n%s", err, b)
					return
				}
				resp = s.handleOffsetRequest(nodeID, conn, req)
			case proto.MetadataReqKind:
				req, err := proto.ReadMetadataReq(bytes.NewBuffer(b))
				if err != nil {
					log.Printf("cannot parse metadata request: %s\n%s", err, b)
					return
				}
				resp = s.handleMetadataRequest(nodeID, conn, req)
			case proto.OffsetCommitReqKind:
				log.Printf("not implemented: %d\n%s", kind, b)
				return
			case proto.OffsetFetchReqKind:
				log.Printf("not implemented: %d\n%s", kind, b)
				return
			case proto.ConsumerMetadataReqKind:
				log.Printf("not implemented: %d\n%s", kind, b)
				return
			default:
				log.Printf("unknown request: %d\n%s", kind, b)
				return
			}
		}

		if resp == nil {
			log.Printf("no response for %d", kind)
			return
		}
		b, err = resp.Bytes()
		if err != nil {
			log.Printf("cannot serialize %T response: %s", resp, err)
		}
		if _, err := conn.Write(b); err != nil {
			log.Printf("cannot write %T response: %s", resp, err)
			return
		}
	}
}

type response interface {
	Bytes() ([]byte, error)
}

func (s *Server) handleProduceRequest(nodeID int32, conn net.Conn, req *proto.ProduceReq) response {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &proto.ProduceResp{
		CorrelationID: req.CorrelationID,
		Topics:        make([]proto.ProduceRespTopic, len(req.Topics)),
	}

	for ti, topic := range req.Topics {
		t, ok := s.topics[topic.Name]
		if !ok {
			t = make(map[int32][]*proto.Message)
			s.topics[topic.Name] = t
		}

		respParts := make([]proto.ProduceRespPartition, len(topic.Partitions))
		resp.Topics[ti].Name = topic.Name
		resp.Topics[ti].Partitions = respParts

		for pi, part := range topic.Partitions {
			p, ok := t[part.ID]
			if !ok {
				p = make([]*proto.Message, 0)
				t[part.ID] = p
			}

			for _, msg := range part.Messages {
				msg.Offset = int64(len(t[part.ID]))
				msg.Topic = topic.Name
				t[part.ID] = append(t[part.ID], msg)
			}

			respParts[pi].ID = part.ID
			respParts[pi].Offset = int64(len(t[part.ID])) - 1
		}
	}
	return resp
}

func (s *Server) handleFetchRequest(nodeID int32, conn net.Conn, req *proto.FetchReq) response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &proto.FetchResp{
		CorrelationID: req.CorrelationID,
		Topics:        make([]proto.FetchRespTopic, len(req.Topics)),
	}
	for ti, topic := range req.Topics {
		respParts := make([]proto.FetchRespPartition, len(topic.Partitions))
		resp.Topics[ti].Name = topic.Name
		resp.Topics[ti].Partitions = respParts
		for pi, part := range topic.Partitions {
			respParts[pi].ID = part.ID

			partitions, ok := s.topics[topic.Name]
			if !ok {
				respParts[pi].Err = proto.ErrUnknownTopicOrPartition
				continue
			}
			messages, ok := partitions[part.ID]
			if !ok {
				respParts[pi].Err = proto.ErrUnknownTopicOrPartition
				continue
			}
			respParts[pi].TipOffset = int64(len(messages))
			respParts[pi].Messages = messages[part.FetchOffset:]
		}
	}

	return resp
}

func (s *Server) handleOffsetRequest(nodeID int32, conn net.Conn, req *proto.OffsetReq) response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &proto.OffsetResp{
		CorrelationID: req.CorrelationID,
		Topics:        make([]proto.OffsetRespTopic, len(req.Topics)),
	}
	for ti, topic := range req.Topics {
		respPart := make([]proto.OffsetRespPartition, len(topic.Partitions))
		resp.Topics[ti].Name = topic.Name
		resp.Topics[ti].Partitions = respPart
		for pi, part := range topic.Partitions {
			respPart[pi].ID = part.ID
			switch part.TimeMs {
			case -1: // oldest
				msgs := len(s.topics[topic.Name][part.ID])
				respPart[pi].Offsets = []int64{int64(msgs), 0}
			case -2: // earliest
				respPart[pi].Offsets = []int64{0, 0}
			default:
				log.Printf("offset time for %s:%d not supported: %d", topic.Name, part.ID, part.TimeMs)
				return nil
			}
		}
	}
	return resp
}

func (s *Server) handleMetadataRequest(nodeID int32, conn net.Conn, req *proto.MetadataReq) response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := &proto.MetadataResp{
		CorrelationID: req.CorrelationID,
		Topics:        make([]proto.MetadataRespTopic, 0, len(s.topics)),
		Brokers:       s.brokers,
	}

	if req.Topics != nil && len(req.Topics) > 0 {
		// if particular topic was requested, create empty log if does not yet exists
		for _, name := range req.Topics {
			partitions, ok := s.topics[name]
			if !ok {
				partitions = make(map[int32][]*proto.Message)
				partitions[0] = make([]*proto.Message, 0)
				s.topics[name] = partitions
			}

			parts := make([]proto.MetadataRespPartition, len(partitions))
			for pid := range partitions {
				p := &parts[pid]
				p.ID = pid
				p.Leader = nodeID
				p.Replicas = []int32{nodeID}
				p.Isrs = []int32{nodeID}
			}
			resp.Topics = append(resp.Topics, proto.MetadataRespTopic{
				Name:       name,
				Partitions: parts,
			})

		}
	} else {
		for name, partitions := range s.topics {
			parts := make([]proto.MetadataRespPartition, len(partitions))
			for pid := range partitions {
				p := &parts[pid]
				p.ID = pid
				p.Leader = nodeID
				p.Replicas = []int32{nodeID}
				p.Isrs = []int32{nodeID}
			}
			resp.Topics = append(resp.Topics, proto.MetadataRespTopic{
				Name:       name,
				Partitions: parts,
			})
		}
	}
	return resp
}

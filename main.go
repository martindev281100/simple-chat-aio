// main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

/***************
 *  Data Models
 ***************/
type User struct {
	AuthKey   string    `json:"authKey"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	LastSeen  time.Time `json:"lastSeen"`
}

type Message struct {
	ID            string        `json:"id"`
	RoomID        string        `json:"roomId"`
	SenderAuthKey string        `json:"sender"`
	Content       string        `json:"content"`
	Timestamp     time.Time     `json:"timestamp"`
	Type          string        `json:"type"` // "text", "system"
	Reactions     ReactionState `json:"-"`    // emoji -> {authKey: true}
	Seen          SeenState     `json:"-"`    // authKey -> seenAt

	// NEW: id of the message this one replies to (if any)
	ReplyToID string `json:"replyTo,omitempty"`

	// NEW: id of the thread this message belongs to (if any)
	ThreadID string `json:"threadId,omitempty"`
}

type Room struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	CreatedBy   string           `json:"createdBy"`
	CreatedAt   time.Time        `json:"createdAt"`
	Members     map[string]bool  `json:"-"`
	Messages    []Message        `json:"-"`
	conns       map[*wsConn]bool `json:"-"`
	mu          sync.RWMutex     `json:"-"`
	maxHist     int              `json:"-"`
}

// Thread represents a Discord-style thread within a room
type Thread struct {
	ID            string           `json:"id"`
	RoomID        string           `json:"roomId"`
	Name          string           `json:"name"`
	RootMessageID string           `json:"rootMessageId,omitempty"` // the message this thread was started from (optional)
	CreatedBy     string           `json:"createdBy"`
	CreatedAt     time.Time        `json:"createdAt"`
	Members       map[string]bool  `json:"-"`
	Messages      []Message        `json:"-"`
	conns         map[*wsConn]bool `json:"-"`
	mu            sync.RWMutex     `json:"-"`
	maxHist       int              `json:"-"`
}

type wsEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// add near other models
type ReactionState map[string]map[string]bool // emoji -> set(authKey)
type SeenState map[string]time.Time           // authKey -> seenAt

type seenSummary struct {
	Count int      `json:"count"`
	Users []string `json:"users,omitempty"` // usernames
}

type replyPreview struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`  // username (not authKey)
	Content   string    `json:"content"` // trimmed
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

/***************
 *  In-Memory Store
 ***************/
type Store struct {
	users   map[string]*User
	rooms   map[string]*Room
	threads map[string]*Thread // NEW
	mu      sync.RWMutex
}

func NewStore() *Store {
	return &Store{
		users:   make(map[string]*User),
		rooms:   make(map[string]*Room),
		threads: make(map[string]*Thread), // NEW
	}
}

func (s *Store) username(authKey string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.users[authKey]; ok && strings.TrimSpace(u.Username) != "" {
		return u.Username
	}
	return authKey
}

func (s *Store) getOrCreateUser(authKey, username string) *User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[authKey]
	if !ok {
		if username == "" {
			username = "user-" + shortID()
		}
		u = &User{
			AuthKey:   authKey,
			Username:  username,
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		s.users[authKey] = u
	} else {
		u.LastSeen = time.Now()
		if username != "" && u.Username == "" {
			u.Username = username
		}
	}
	return u
}

func (s *Store) updateUsername(authKey, username string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[authKey]
	if !ok {
		return nil, errors.New("user not found")
	}
	u.Username = username
	u.LastSeen = time.Now()
	return u, nil
}

func (s *Store) listRooms() []*Room {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Room, 0, len(s.rooms))
	for _, r := range s.rooms {
		out = append(out, r)
	}
	return out
}

func (s *Store) createRoom(name, desc, createdBy string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := "room-" + shortID()
	r := &Room{
		ID:          id,
		Name:        name,
		Description: desc,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		Members:     map[string]bool{},
		Messages:    []Message{},
		conns:       map[*wsConn]bool{},
		maxHist:     200,
	}
	s.rooms[id] = r
	return r
}

func (s *Store) getRoom(id string) (*Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rooms[id]
	return r, ok
}

// createThread creates a new thread in a room, optionally from a root message
func (s *Store) createThread(roomID, name, createdBy, rootMessageID string) (*Thread, error) {
	s.mu.RLock()
	room, ok := s.rooms[roomID]
	s.mu.RUnlock()
	if !ok {
		return nil, errors.New("room not found")
	}

	th := &Thread{
		ID:        "thr-" + shortID(),
		RoomID:    roomID,
		Name:      firstNonEmpty(strings.TrimSpace(name), "Thread-"+shortID()),
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
		Members:   map[string]bool{},
		Messages:  []Message{},
		conns:     map[*wsConn]bool{},
		maxHist:   200,
	}

	// If created from a message, verify it exists and annotate it with ThreadID
	if strings.TrimSpace(rootMessageID) != "" {
		room.mu.Lock()
		msg := (&app{}).findMessage(room, rootMessageID)
		if msg == nil {
			room.mu.Unlock()
			return nil, errors.New("root message not found")
		}
		th.RootMessageID = rootMessageID
		// Mark the root message as having a thread pointer
		msg.ThreadID = th.ID
		room.mu.Unlock()
	}

	s.mu.Lock()
	s.threads[th.ID] = th
	s.mu.Unlock()
	return th, nil
}

// getThread retrieves a thread by ID
func (s *Store) getThread(threadID string) (*Thread, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	th, ok := s.threads[threadID]
	return th, ok
}

// listThreads returns all threads in a room
func (s *Store) listThreads(roomID string) []*Thread {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []*Thread{}
	for _, th := range s.threads {
		if th.RoomID == roomID {
			out = append(out, th)
		}
	}
	return out
}

/***************
 *  Auth (Gin)
 ***************/
const ginUserKey = "user"

func authMiddleware(store *Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1) Lấy từ header trước
		authKey := strings.TrimSpace(c.GetHeader("X-Auth-Key"))
		username := strings.TrimSpace(c.GetHeader("X-Username"))

		// 2) Nếu thiếu -> fallback sang query params (hữu ích cho WebSocket browsers)
		if authKey == "" {
			authKey = strings.TrimSpace(c.Query("authKey"))
		}
		if username == "" {
			username = strings.TrimSpace(c.Query("username"))
		}

		if authKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth (X-Auth-Key header or ?authKey=... query)"})
			return
		}

		user := store.getOrCreateUser(authKey, username)
		c.Set(ginUserKey, user)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ginUserKey, user))
		c.Next()
	}
}

func getUser(c *gin.Context) *User {
	v, ok := c.Get(ginUserKey)
	if !ok {
		return nil
	}
	u, _ := v.(*User)
	return u
}

/***************
 *  HTTP Handlers (Gin)
 ***************/
type app struct {
	store    *Store
	upgrader websocket.Upgrader
}

// reactionResp is sent to clients
type reactionResp struct {
	Emoji    string   `json:"emoji"`
	Count    int      `json:"count"`
	Reactors []string `json:"reactors,omitempty"` // usernames
}

// summarize reactions for one message
func (a *app) summarizeReactions(m *Message) []reactionResp {
	if m.Reactions == nil {
		return nil
	}
	out := make([]reactionResp, 0, len(m.Reactions))
	for emoji, set := range m.Reactions {
		count := len(set)
		if count == 0 {
			continue
		}
		reactors := make([]string, 0, count)
		for ak := range set {
			reactors = append(reactors, a.store.username(ak))
		}
		out = append(out, reactionResp{
			Emoji:    emoji,
			Count:    count,
			Reactors: reactors,
		})
	}
	return out
}

func (a *app) summarizeSeen(m *Message) seenSummary {
	if m.Seen == nil || len(m.Seen) == 0 {
		return seenSummary{Count: 0, Users: nil}
	}
	users := make([]string, 0, len(m.Seen))
	for ak := range m.Seen {
		users = append(users, a.store.username(ak))
	}
	return seenSummary{Count: len(users), Users: users}
}

func (a *app) makeReplyPreview(room *Room, m *Message) *replyPreview {
	if m == nil {
		return nil
	}
	return &replyPreview{
		ID:        m.ID,
		Sender:    a.store.username(m.SenderAuthKey),
		Content:   trimRunes(m.Content, 120),
		Timestamp: m.Timestamp,
		Type:      m.Type,
	}
}

// requires caller to hold at least RLock on room.mu (or accept a brief RLock here)
func (a *app) replyPreviewByID(room *Room, id string) *replyPreview {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	// we can RLock here for safety if caller doesn't hold a lock
	room.mu.RLock()
	defer room.mu.RUnlock()
	m := a.findMessage(room, id)
	return a.makeReplyPreview(room, m)
}

// find message pointer by id (must be called under room.mu lock or rlock accordingly)
func (a *app) findMessage(room *Room, messageID string) *Message {
	for i := range room.Messages {
		if room.Messages[i].ID == messageID {
			return &room.Messages[i]
		}
	}
	return nil
}

// find message index by id (must be called under room.mu lock)
func (a *app) findMessageIndex(room *Room, messageID string) int {
	for i := range room.Messages {
		if room.Messages[i].ID == messageID {
			return i
		}
	}
	return -1
}

// setReaction mutates a message reaction set (add/remove). Returns (changed, count, reactors, err)
func (a *app) setReaction(room *Room, messageID, emoji, authKey string, add bool) (bool, int, []string, error) {
	if strings.TrimSpace(emoji) == "" {
		return false, 0, nil, errors.New("emoji required")
	}
	room.mu.Lock()
	defer room.mu.Unlock()

	msg := a.findMessage(room, messageID)
	if msg == nil {
		return false, 0, nil, errors.New("message not found")
	}
	if msg.Reactions == nil {
		msg.Reactions = make(ReactionState)
	}
	if _, ok := msg.Reactions[emoji]; !ok {
		msg.Reactions[emoji] = make(map[string]bool)
	}

	changed := false
	if add {
		if !msg.Reactions[emoji][authKey] {
			msg.Reactions[emoji][authKey] = true
			changed = true
		}
	} else {
		if msg.Reactions[emoji][authKey] {
			delete(msg.Reactions[emoji], authKey)
			changed = true
			if len(msg.Reactions[emoji]) == 0 {
				delete(msg.Reactions, emoji)
			}
		}
	}

	// Build summary for response
	var count int
	var reactors []string
	if set, ok := msg.Reactions[emoji]; ok {
		count = len(set)
		reactors = make([]string, 0, count)
		for ak := range set {
			reactors = append(reactors, a.store.username(ak))
		}
	} else {
		count = 0
	}

	return changed, count, reactors, nil
}

// broadcast one reaction change
func (a *app) broadcastReaction(room *Room, messageID, emoji string, count int, reactors []string) {
	payload, _ := json.Marshal(map[string]any{
		"roomId":    room.ID,
		"messageId": messageID,
		"emoji":     emoji,
		"count":     count,
		"reactors":  reactors, // usernames
	})
	a.broadcast(room, wsEnvelope{
		Event: "reaction_updated",
		Data:  payload,
	}, nil)
}

// setSeen marks a message as seen by authKey.
// Returns (changed, usersForTarget, countForTarget, err)
func (a *app) setSeen(room *Room, messageID, authKey string) (bool, []string, int, error) {
	now := time.Now()
	room.mu.Lock()
	defer room.mu.Unlock()

	idx := a.findMessageIndex(room, messageID)
	if idx == -1 {
		return false, nil, 0, errors.New("message not found")
	}

	msg := &room.Messages[idx]
	if msg.Seen == nil {
		msg.Seen = make(SeenState)
	}

	prev, ok := msg.Seen[authKey]
	msg.Seen[authKey] = now
	changed := !ok || prev.Before(now)

	users := make([]string, 0, len(msg.Seen))
	for ak := range msg.Seen {
		users = append(users, a.store.username(ak))
	}

	return changed, users, len(users), nil
}

// broadcast one seen update
func (a *app) broadcastSeen(room *Room, messageID string, count int, users []string) {
	payload, _ := json.Marshal(map[string]any{
		"roomId":    room.ID,
		"messageId": messageID,
		"count":     count,
		"users":     users, // usernames
	})
	a.broadcast(room, wsEnvelope{
		Event: "seen_updated",
		Data:  payload,
	}, nil)
}

func newApp() *app {
	allowedOrigins := []string{
		"http://127.0.0.1:5500",
		"http://localhost:5500",
		"http://localhost:8080",
	}
	return &app{
		store: NewStore(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}
				// allow non-browser clients (no Origin)
				return origin == ""
			},
		},
	}
}

// --- User
func (a *app) getProfile(c *gin.Context) {
	u := getUser(c)
	c.JSON(http.StatusOK, u)
}

func (a *app) updateProfile(c *gin.Context) {
	u := getUser(c)
	var payload struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
		return
	}
	upd, err := a.store.updateUsername(u.AuthKey, strings.TrimSpace(payload.Username))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update profile"})
		return
	}
	c.JSON(http.StatusOK, upd)
}

// --- Rooms
func (a *app) listRooms(c *gin.Context) {
	type roomItem struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
		MemberCount int    `json:"memberCount"`
	}
	rooms := a.store.listRooms()
	resp := make([]roomItem, 0, len(rooms))
	for _, room := range rooms {
		room.mu.RLock()
		memberCount := len(room.Members)
		room.mu.RUnlock()
		resp = append(resp, roomItem{
			ID:          room.ID,
			Name:        room.Name,
			Description: room.Description,
			MemberCount: memberCount,
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (a *app) createRoom(c *gin.Context) {
	u := getUser(c)
	var payload struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	room := a.store.createRoom(strings.TrimSpace(payload.Name), strings.TrimSpace(payload.Description), u.AuthKey)
	c.JSON(http.StatusOK, room)
}

func (a *app) getRoom(c *gin.Context) {
	id := c.Param("roomId")
	room, ok := a.store.getRoom(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	type resp struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description,omitempty"`
		CreatedBy   string    `json:"createdBy"`
		CreatedAt   time.Time `json:"createdAt"`
		MemberCount int       `json:"memberCount"`
	}
	room.mu.RLock()
	defer room.mu.RUnlock()
	c.JSON(http.StatusOK, resp{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		CreatedBy:   room.CreatedBy,
		CreatedAt:   room.CreatedAt,
		MemberCount: len(room.Members),
	})
}

func (a *app) joinRoomHTTP(c *gin.Context) {
	u := getUser(c)
	id := c.Param("roomId")
	room, ok := a.store.getRoom(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	room.mu.Lock()
	room.Members[u.AuthKey] = true
	online := a.roomOnlineUsernames(room)
	room.mu.Unlock()

	a.addSystemMessage(room, u.Username+" joined the room")

	c.JSON(http.StatusOK, gin.H{
		"roomId":      room.ID,
		"roomName":    room.Name,
		"onlineUsers": online,
	})
}

func (a *app) leaveRoomHTTP(c *gin.Context) {
	u := getUser(c)
	id := c.Param("roomId")
	room, ok := a.store.getRoom(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	room.mu.Lock()
	delete(room.Members, u.AuthKey)
	room.mu.Unlock()

	a.addSystemMessage(room, u.Username+" left the room")
	c.JSON(http.StatusOK, gin.H{"roomId": room.ID})
}

// --- Messages
func (a *app) getMessages(c *gin.Context) {
	id := c.Param("roomId")
	room, ok := a.store.getRoom(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	limit := clamp(parseInt(c.Query("limit"), 50), 1, 200)
	offset := clamp(parseInt(c.Query("offset"), 0), 0, 100000)

	type messageResp struct {
		ID        string         `json:"id"`
		RoomID    string         `json:"roomId"`
		Sender    string         `json:"sender"` // authKey (kept compat with old API)
		Content   string         `json:"content"`
		Timestamp time.Time      `json:"timestamp"`
		Type      string         `json:"type"`
		Reactions []reactionResp `json:"reactions,omitempty"`
		Seen      seenSummary    `json:"seen"`

		// NEW
		ReplyTo *replyPreview `json:"replyTo,omitempty"`
	}

	room.mu.RLock()
	defer room.mu.RUnlock()

	n := len(room.Messages)
	start := offset
	if start > n {
		start = n
	}
	end := start + limit
	if end > n {
		end = n
	}

	out := make([]messageResp, 0, end-start)
	for i := start; i < end; i++ {
		m := &room.Messages[i]
		var rp *replyPreview
		if m.ReplyToID != "" {
			rp = a.replyPreviewByID(room, m.ReplyToID)
		}
		out = append(out, messageResp{
			ID:        m.ID,
			RoomID:    m.RoomID,
			Sender:    m.SenderAuthKey, // unchanged
			Content:   m.Content,
			Timestamp: m.Timestamp,
			Type:      m.Type,
			Reactions: a.summarizeReactions(m),
			Seen:      a.summarizeSeen(m),
			ReplyTo:   rp, // NEW
		})
	}
	c.JSON(http.StatusOK, out)
}

func (a *app) addReactionHTTP(c *gin.Context) {
	u := getUser(c)
	roomID := c.Param("roomId")
	messageID := c.Param("messageId")
	emoji := c.Param("emoji")

	room, ok := a.store.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	changed, count, reactors, err := a.setReaction(room, messageID, emoji, u.AuthKey, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if changed {
		a.broadcastReaction(room, messageID, emoji, count, reactors)
	}
	c.JSON(http.StatusOK, gin.H{
		"roomId":    room.ID,
		"messageId": messageID,
		"emoji":     emoji,
		"count":     count,
		"reactors":  reactors,
	})
}

func (a *app) removeReactionHTTP(c *gin.Context) {
	u := getUser(c)
	roomID := c.Param("roomId")
	messageID := c.Param("messageId")
	emoji := c.Param("emoji")

	room, ok := a.store.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	changed, count, reactors, err := a.setReaction(room, messageID, emoji, u.AuthKey, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if changed {
		a.broadcastReaction(room, messageID, emoji, count, reactors)
	}
	c.JSON(http.StatusOK, gin.H{
		"roomId":    room.ID,
		"messageId": messageID,
		"emoji":     emoji,
		"count":     count,
		"reactors":  reactors,
	})
}

// --- Threads (HTTP)
func (a *app) listThreads(c *gin.Context) {
	roomID := c.Param("roomId")
	ths := a.store.listThreads(roomID)
	type item struct {
		ID            string    `json:"id"`
		RoomID        string    `json:"roomId"`
		Name          string    `json:"name"`
		RootMessageID string    `json:"rootMessageId,omitempty"`
		CreatedBy     string    `json:"createdBy"`
		CreatedAt     time.Time `json:"createdAt"`
		MemberCount   int       `json:"memberCount"`
	}
	out := make([]item, 0, len(ths))
	for _, th := range ths {
		th.mu.RLock()
		out = append(out, item{
			ID:            th.ID,
			RoomID:        th.RoomID,
			Name:          th.Name,
			RootMessageID: th.RootMessageID,
			CreatedBy:     th.CreatedBy,
			CreatedAt:     th.CreatedAt,
			MemberCount:   len(th.Members),
		})
		th.mu.RUnlock()
	}
	c.JSON(http.StatusOK, out)
}

func (a *app) createThreadHTTP(c *gin.Context) {
	u := getUser(c)
	roomID := c.Param("roomId")
	var p struct {
		Name          string `json:"name"`
		RootMessageID string `json:"rootMessageId"` // optional
	}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
		return
	}
	th, err := a.store.createThread(roomID, p.Name, u.AuthKey, strings.TrimSpace(p.RootMessageID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Broadcast to room
	if room, ok := a.store.getRoom(roomID); ok {
		payload, _ := json.Marshal(map[string]any{
			"threadId":      th.ID,
			"roomId":        th.RoomID,
			"name":          th.Name,
			"rootMessageId": th.RootMessageID,
			"createdBy":     a.store.username(u.AuthKey),
			"createdAt":     th.CreatedAt,
		})
		a.broadcast(room, wsEnvelope{Event: "new_thread", Data: payload}, nil)
	}
	c.JSON(http.StatusOK, th)
}

func (a *app) getThread(c *gin.Context) {
	threadID := c.Param("threadId")
	th, ok := a.store.getThread(threadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "thread not found"})
		return
	}
	th.mu.RLock()
	defer th.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"id":            th.ID,
		"roomId":        th.RoomID,
		"name":          th.Name,
		"rootMessageId": th.RootMessageID,
		"createdBy":     th.CreatedBy,
		"createdAt":     th.CreatedAt,
		"memberCount":   len(th.Members),
	})
}

func (a *app) getThreadMessages(c *gin.Context) {
	threadID := c.Param("threadId")
	th, ok := a.store.getThread(threadID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "thread not found"})
		return
	}
	limit := clamp(parseInt(c.Query("limit"), 50), 1, 200)
	offset := clamp(parseInt(c.Query("offset"), 0), 0, 100000)

	type messageResp struct {
		ID        string         `json:"id"`
		RoomID    string         `json:"roomId"`
		ThreadID  string         `json:"threadId"`
		Sender    string         `json:"sender"`
		Content   string         `json:"content"`
		Timestamp time.Time      `json:"timestamp"`
		Type      string         `json:"type"`
		Reactions []reactionResp `json:"reactions,omitempty"`
		Seen      seenSummary    `json:"seen"`
		ReplyTo   *replyPreview  `json:"replyTo,omitempty"`
	}

	th.mu.RLock()
	defer th.mu.RUnlock()

	n := len(th.Messages)
	start := offset
	if start > n {
		start = n
	}
	end := start + limit
	if end > n {
		end = n
	}

	out := make([]messageResp, 0, end-start)
	for i := start; i < end; i++ {
		m := &th.Messages[i]
		var rp *replyPreview
		if m.ReplyToID != "" {
			// look up within thread
			rp = a.makeReplyPreviewForThread(th, a.findThreadMessage(th, m.ReplyToID))
		}
		out = append(out, messageResp{
			ID:        m.ID,
			RoomID:    m.RoomID,
			ThreadID:  m.ThreadID,
			Sender:    m.SenderAuthKey,
			Content:   m.Content,
			Timestamp: m.Timestamp,
			Type:      m.Type,
			Reactions: a.summarizeReactions(m),
			Seen:      a.summarizeSeen(m),
			ReplyTo:   rp,
		})
	}
	c.JSON(http.StatusOK, out)
}

// --- Seen (HTTP)
func (a *app) markSeenHTTP(c *gin.Context) {
	u := getUser(c)
	roomID := c.Param("roomId")
	messageID := c.Param("messageId")

	room, ok := a.store.getRoom(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	changed, users, count, err := a.setSeen(room, messageID, u.AuthKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if changed {
		a.broadcastSeen(room, messageID, count, users)
	}
	c.JSON(http.StatusOK, gin.H{
		"roomId":    room.ID,
		"messageId": messageID,
		"count":     count,
		"users":     users,
	})
}

/***************
 *  WebSocket
 ***************/
type wsConn struct {
	conn     *websocket.Conn
	writeMu  sync.Mutex
	authKey  string
	username string
	roomID   string
	threadID string // NEW
	app      *app
}

func (a *app) wsHandler(c *gin.Context) {
	u := getUser(c)
	if u == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	ws, err := a.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}
	client := &wsConn{
		conn:     ws,
		authKey:  u.AuthKey,
		username: firstNonEmpty(u.Username, "user-"+shortID()),
		app:      a,
	}
	go client.readLoop()
}

func (c *wsConn) readLoop() {
	defer func() {
		c.cleanup()
		c.conn.Close()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var env wsEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			c.sendError("invalid_envelope", "bad json")
			continue
		}
		switch env.Event {
		case "join_room":
			var payload struct {
				RoomID string `json:"roomId"`
			}
			if err := json.Unmarshal(env.Data, &payload); err != nil || strings.TrimSpace(payload.RoomID) == "" {
				c.sendError("join_room_error", "roomId required")
				continue
			}
			c.joinRoom(strings.TrimSpace(payload.RoomID))
		case "leave_room":
			c.leaveRoom()
		case "send_message":
			var payload struct {
				RoomID  string `json:"roomId"`
				Content string `json:"content"`
				ReplyTo string `json:"replyTo"` // NEW (optional)
			}
			if err := json.Unmarshal(env.Data, &payload); err != nil {
				c.sendError("send_message_error", "bad json")
				continue
			}
			c.handleSendMessage(strings.TrimSpace(payload.RoomID), strings.TrimSpace(payload.Content), strings.TrimSpace(payload.ReplyTo))
		case "add_reaction":
			var p struct {
				RoomID    string `json:"roomId"`
				MessageID string `json:"messageId"`
				Emoji     string `json:"emoji"`
			}
			if err := json.Unmarshal(env.Data, &p); err != nil {
				c.sendError("add_reaction_error", "bad json")
				continue
			}
			c.handleReaction(p.RoomID, p.MessageID, p.Emoji, true)
		case "remove_reaction":
			var p struct {
				RoomID    string `json:"roomId"`
				MessageID string `json:"messageId"`
				Emoji     string `json:"emoji"`
			}
			if err := json.Unmarshal(env.Data, &p); err != nil {
				c.sendError("remove_reaction_error", "bad json")
				continue
			}
			c.handleReaction(p.RoomID, p.MessageID, p.Emoji, false)
		case "toggle_reaction":
			var p struct {
				RoomID    string `json:"roomId"`
				MessageID string `json:"messageId"`
				Emoji     string `json:"emoji"`
			}
			if err := json.Unmarshal(env.Data, &p); err != nil {
				c.sendError("toggle_reaction_error", "bad json")
				continue
			}
			c.handleToggleReaction(p.RoomID, p.MessageID, p.Emoji)
		case "mark_seen":
			var p struct {
				RoomID    string `json:"roomId"`
				MessageID string `json:"messageId"`
			}
			if err := json.Unmarshal(env.Data, &p); err != nil {
				c.sendError("mark_seen_error", "bad json")
				continue
			}
			c.handleMarkSeen(p.RoomID, p.MessageID)
		case "create_thread":
			// create a new thread (optionally from a message)
			var p struct {
				RoomID        string `json:"roomId"`
				Name          string `json:"name"`
				RootMessageID string `json:"rootMessageId"` // optional
			}
			if err := json.Unmarshal(env.Data, &p); err != nil {
				c.sendError("create_thread_error", "bad json")
				continue
			}
			c.handleCreateThread(p.RoomID, strings.TrimSpace(p.Name), strings.TrimSpace(p.RootMessageID))
		case "join_thread":
			var p struct {
				ThreadID string `json:"threadId"`
			}
			if err := json.Unmarshal(env.Data, &p); err != nil || strings.TrimSpace(p.ThreadID) == "" {
				c.sendError("join_thread_error", "threadId required")
				continue
			}
			c.joinThread(strings.TrimSpace(p.ThreadID))
		case "leave_thread":
			c.leaveThread()
		case "send_thread_message":
			var p struct {
				ThreadID string `json:"threadId"`
				Content  string `json:"content"`
				ReplyTo  string `json:"replyTo"`
			}
			if err := json.Unmarshal(env.Data, &p); err != nil {
				c.sendError("send_thread_message_error", "bad json")
				continue
			}
			c.handleSendThreadMessage(strings.TrimSpace(p.ThreadID), strings.TrimSpace(p.Content), strings.TrimSpace(p.ReplyTo))
		default:
			c.sendError("unknown_event", "unsupported event: "+env.Event)
		}
	}
}

func (c *wsConn) joinRoom(roomID string) {
	room, ok := c.app.store.getRoom(roomID)
	if !ok {
		c.sendError("join_room_error", "room not found")
		return
	}

	room.mu.Lock()
	room.Members[c.authKey] = true
	room.conns[c] = true
	c.roomID = room.ID
	online := c.app.roomOnlineUsernames(room)
	roomName := room.Name
	room.mu.Unlock()

	data := map[string]any{
		"roomId":      room.ID,
		"roomName":    roomName,
		"onlineUsers": online,
	}
	jsonData, _ := json.Marshal(data)
	c.sendJSON(wsEnvelope{
		Event: "room_joined",
		Data:  jsonData,
	})

	broadCastData := map[string]string{
		"roomId":   room.ID,
		"username": c.username,
	}
	jsonDataBroad, _ := json.Marshal(broadCastData)
	c.app.broadcast(room, wsEnvelope{
		Event: "user_joined",
		Data:  jsonDataBroad,
	}, skipConn(c))
}

func (c *wsConn) leaveRoom() {
	if c.roomID == "" {
		return
	}
	room, ok := c.app.store.getRoom(c.roomID)
	if !ok {
		return
	}
	room.mu.Lock()
	delete(room.conns, c)
	delete(room.Members, c.authKey)
	room.mu.Unlock()

	data := map[string]string{"roomId": c.roomID}
	jsonData, _ := json.Marshal(data)
	c.app.broadcast(room, wsEnvelope{
		Event: "user_left",
		Data:  jsonData,
	}, nil)
	c.roomID = ""
}

func (c *wsConn) handleSendMessage(roomID, content, replyTo string) {
	if content == "" {
		c.sendError("send_message_error", "content required")
		return
	}
	// must be in the room (auto-join like before)
	if c.roomID == "" || c.roomID != roomID {
		c.joinRoom(roomID)
		if c.roomID != roomID {
			return
		}
	}
	room, ok := c.app.store.getRoom(roomID)
	if !ok {
		c.sendError("send_message_error", "room not found")
		return
	}

	// Validate reply target (if provided)
	var rp *replyPreview
	if replyTo != "" {
		room.mu.RLock()
		target := c.app.findMessage(room, replyTo)
		if target == nil {
			room.mu.RUnlock()
			c.sendError("send_message_error", "replyTo message not found in this room")
			return
		}
		rp = c.app.makeReplyPreview(room, target)
		room.mu.RUnlock()
	}

	msg := Message{
		ID:            "msg-" + shortID(),
		RoomID:        room.ID,
		SenderAuthKey: c.authKey,
		Content:       content,
		Timestamp:     time.Now(),
		Type:          "text",
		Seen:          SeenState{c.authKey: time.Now()}, // sender has seen
		ReplyToID:     replyTo,                          // store linkage
	}

	room.mu.Lock()
	room.Messages = append(room.Messages, msg)
	if len(room.Messages) > room.maxHist {
		room.Messages = room.Messages[len(room.Messages)-room.maxHist:]
	}
	room.mu.Unlock()

	data := map[string]any{
		"id":        msg.ID,
		"roomId":    msg.RoomID,
		"sender":    c.username, // username in WS broadcasts (unchanged)
		"content":   msg.Content,
		"timestamp": msg.Timestamp,
		"reactions": []reactionResp{},
		"seen":      seenSummary{Count: 1, Users: []string{c.username}},
		"replyTo":   rp, // NEW – already a preview struct or nil
	}
	payload, _ := json.Marshal(data)
	c.app.broadcast(room, wsEnvelope{
		Event: "new_message",
		Data:  payload,
	}, nil)
}

func (c *wsConn) handleReaction(roomID, messageID, emoji string, add bool) {
	if strings.TrimSpace(emoji) == "" {
		c.sendError("reaction_error", "emoji required")
		return
	}
	// must be in the room (auto-join if necessary, like send_message)
	if c.roomID == "" || c.roomID != roomID {
		c.joinRoom(roomID)
		if c.roomID != roomID {
			return
		}
	}
	room, ok := c.app.store.getRoom(roomID)
	if !ok {
		c.sendError("reaction_error", "room not found")
		return
	}

	changed, count, reactors, err := c.app.setReaction(room, messageID, emoji, c.authKey, add)
	if err != nil {
		c.sendError("reaction_error", err.Error())
		return
	}
	if changed {
		c.app.broadcastReaction(room, messageID, emoji, count, reactors)
	}
}

func (c *wsConn) handleToggleReaction(roomID, messageID, emoji string) {
	// try add first; if no change -> remove
	if c.roomID == "" || c.roomID != roomID {
		c.joinRoom(roomID)
		if c.roomID != roomID {
			return
		}
	}
	room, ok := c.app.store.getRoom(roomID)
	if !ok {
		c.sendError("reaction_error", "room not found")
		return
	}
	changed, count, reactors, err := c.app.setReaction(room, messageID, emoji, c.authKey, true)
	if err != nil {
		c.sendError("reaction_error", err.Error())
		return
	}
	if changed {
		c.app.broadcastReaction(room, messageID, emoji, count, reactors)
		return
	}
	// already had it -> remove
	changed, count, reactors, err = c.app.setReaction(room, messageID, emoji, c.authKey, false)
	if err != nil {
		c.sendError("reaction_error", err.Error())
		return
	}
	if changed {
		c.app.broadcastReaction(room, messageID, emoji, count, reactors)
	}
}

func (c *wsConn) handleMarkSeen(roomID, messageID string) {
	if c.roomID == "" || c.roomID != roomID {
		c.joinRoom(roomID)
		if c.roomID != roomID {
			return
		}
	}
	room, ok := c.app.store.getRoom(roomID)
	if !ok {
		c.sendError("mark_seen_error", "room not found")
		return
	}
	changed, users, count, err := c.app.setSeen(room, messageID, c.authKey)
	if err != nil {
		c.sendError("mark_seen_error", err.Error())
		return
	}
	if changed {
		c.app.broadcastSeen(room, messageID, count, users)
	}
}

// handleCreateThread processes thread creation requests
func (c *wsConn) handleCreateThread(roomID, name, rootMessageID string) {
	// ensure user is (or becomes) in room
	if c.roomID == "" || c.roomID != roomID {
		c.joinRoom(roomID)
		if c.roomID != roomID {
			return
		}
	}
	th, err := c.app.store.createThread(roomID, name, c.authKey, rootMessageID)
	if err != nil {
		c.sendError("create_thread_error", err.Error())
		return
	}

	// broadcast to room: new thread created
	payload, _ := json.Marshal(map[string]any{
		"threadId":      th.ID,
		"roomId":        th.RoomID,
		"name":          th.Name,
		"rootMessageId": th.RootMessageID,
		"createdBy":     c.username,
		"createdAt":     th.CreatedAt,
	})
	room, _ := c.app.store.getRoom(roomID)
	c.app.broadcast(room, wsEnvelope{Event: "new_thread", Data: payload}, nil)

	// Optionally, auto-join creator into the thread
	c.joinThread(th.ID)
}

// joinThread joins a user to a thread
func (c *wsConn) joinThread(threadID string) {
	th, ok := c.app.store.getThread(threadID)
	if !ok {
		c.sendError("join_thread_error", "thread not found")
		return
	}
	// ensure in parent room
	if c.roomID == "" || c.roomID != th.RoomID {
		c.joinRoom(th.RoomID)
		if c.roomID != th.RoomID {
			return
		}
	}

	th.mu.Lock()
	th.Members[c.authKey] = true
	th.conns[c] = true
	c.threadID = th.ID
	online := c.app.threadOnlineUsernames(th)
	name := th.Name
	th.mu.Unlock()

	data := map[string]any{
		"threadId":    th.ID,
		"threadName":  name,
		"roomId":      th.RoomID,
		"onlineUsers": online,
	}
	jd, _ := json.Marshal(data)
	c.sendJSON(wsEnvelope{Event: "thread_joined", Data: jd})

	announce, _ := json.Marshal(map[string]string{"threadId": th.ID, "username": c.username})
	c.app.broadcastThread(th, wsEnvelope{Event: "user_joined_thread", Data: announce}, skipConn(c))
}

// leaveThread removes user from current thread
func (c *wsConn) leaveThread() {
	if c.threadID == "" {
		return
	}
	th, ok := c.app.store.getThread(c.threadID)
	if !ok {
		c.threadID = ""
		return
	}
	th.mu.Lock()
	delete(th.conns, c)
	delete(th.Members, c.authKey)
	id := th.ID
	th.mu.Unlock()

	data := map[string]string{"threadId": id}
	jd, _ := json.Marshal(data)
	c.app.broadcastThread(th, wsEnvelope{Event: "user_left_thread", Data: jd}, nil)
	c.threadID = ""
}

// handleSendThreadMessage processes messages sent to threads
func (c *wsConn) handleSendThreadMessage(threadID, content, replyTo string) {
	if content == "" {
		c.sendError("send_thread_message_error", "content required")
		return
	}
	th, ok := c.app.store.getThread(threadID)
	if !ok {
		c.sendError("send_thread_message_error", "thread not found")
		return
	}
	// auto-join if needed
	if c.threadID == "" || c.threadID != threadID {
		c.joinThread(threadID)
		if c.threadID != threadID {
			return
		}
	}
	// Validate reply target within thread if provided
	var rp *replyPreview
	if replyTo != "" {
		th.mu.RLock()
		target := c.app.findThreadMessage(th, replyTo)
		if target == nil {
			th.mu.RUnlock()
			c.sendError("send_thread_message_error", "replyTo message not found in this thread")
			return
		}
		rp = c.app.makeReplyPreviewForThread(th, target)
		th.mu.RUnlock()
	}

	msg := Message{
		ID:            "msg-" + shortID(),
		RoomID:        th.RoomID,
		ThreadID:      th.ID,
		SenderAuthKey: c.authKey,
		Content:       content,
		Timestamp:     time.Now(),
		Type:          "text",
		Seen:          SeenState{c.authKey: time.Now()},
		ReplyToID:     replyTo,
	}

	th.mu.Lock()
	th.Messages = append(th.Messages, msg)
	if len(th.Messages) > th.maxHist {
		th.Messages = th.Messages[len(th.Messages)-th.maxHist:]
	}
	th.mu.Unlock()

	payload, _ := json.Marshal(map[string]any{
		"id":        msg.ID,
		"roomId":    msg.RoomID,
		"threadId":  msg.ThreadID,
		"sender":    c.username,
		"content":   msg.Content,
		"timestamp": msg.Timestamp,
		"reactions": []reactionResp{},
		"seen":      seenSummary{Count: 1, Users: []string{c.username}},
		"replyTo":   rp,
	})
	c.app.broadcastThread(th, wsEnvelope{Event: "new_thread_message", Data: payload}, nil)
}

// threadOnlineUsernames returns online usernames in a thread
func (a *app) threadOnlineUsernames(th *Thread) []string {
	names := []string{}
	for c := range th.conns {
		names = append(names, c.username)
	}
	return names
}

// broadcastThread sends a message to all connections in a thread
func (a *app) broadcastThread(th *Thread, env wsEnvelope, skip func(*wsConn) bool) {
	th.mu.RLock()
	defer th.mu.RUnlock()
	for conn := range th.conns {
		if skip != nil && skip(conn) {
			continue
		}
		conn.sendJSON(env)
	}
}

// findThreadMessage finds a message within a thread by ID
func (a *app) findThreadMessage(th *Thread, messageID string) *Message {
	for i := range th.Messages {
		if th.Messages[i].ID == messageID {
			return &th.Messages[i]
		}
	}
	return nil
}

// makeReplyPreviewForThread creates a reply preview for thread messages
func (a *app) makeReplyPreviewForThread(th *Thread, m *Message) *replyPreview {
	if m == nil {
		return nil
	}
	return &replyPreview{
		ID:        m.ID,
		Sender:    a.store.username(m.SenderAuthKey),
		Content:   trimRunes(m.Content, 120),
		Timestamp: m.Timestamp,
		Type:      m.Type,
	}
}

func (c *wsConn) sendJSON(v any) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.WriteJSON(v)
}

func (c *wsConn) sendError(code, message string) {
	payload, _ := json.Marshal(map[string]string{"error": message})
	c.sendJSON(wsEnvelope{
		Event: code,
		Data:  payload,
	})
}

func (c *wsConn) cleanup() {
	if c.threadID != "" {
		c.leaveThread()
	}
	if c.roomID != "" {
		c.leaveRoom()
	}
}

/***************
 *  Broadcast Helpers
 ***************/
func (a *app) broadcast(room *Room, env wsEnvelope, skip func(*wsConn) bool) {
	room.mu.RLock()
	defer room.mu.RUnlock()
	for conn := range room.conns {
		if skip != nil && skip(conn) {
			continue
		}
		conn.sendJSON(env)
	}
}

func skipConn(x *wsConn) func(*wsConn) bool {
	return func(c *wsConn) bool { return c == x }
}

func (a *app) roomOnlineUsernames(room *Room) []string {
	names := []string{}
	for c := range room.conns {
		names = append(names, c.username)
	}
	return names
}

func (a *app) addSystemMessage(room *Room, content string) {
	room.mu.Lock()
	defer room.mu.Unlock()
	msg := Message{
		ID:            "sys-" + shortID(),
		RoomID:        room.ID,
		SenderAuthKey: "system",
		Content:       content,
		Timestamp:     time.Now(),
		Type:          "system",
		Seen:          SeenState{}, // Empty seen state for system messages
	}
	room.Messages = append(room.Messages, msg)
	if len(room.Messages) > room.maxHist {
		room.Messages = room.Messages[len(room.Messages)-room.maxHist:]
	}
}

/***************
 *  Utilities
 ***************/
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return i
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func shortID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatInt(int64(rand.Intn(1e6)), 36)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func trimRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

/***************
 *  Main (Gin + gin-contrib/cors)
 ***************/
func main() {
	rand.Seed(time.Now().UnixNano())

	app := newApp()

	// Seed a default room for convenience
	defaultRoom := app.store.createRoom("General", "Default room", "system")
	log.Println("Seeded room:", defaultRoom.ID, defaultRoom.Name)

	allowedOrigins := []string{
		"http://127.0.0.1:5500",
		"http://localhost:5500",
		"http://localhost:8080",
	}

	g := gin.New()
	g.Use(gin.Logger(), gin.Recovery())

	// CORS via gin-contrib/cors
	g.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Auth-Key", "X-Username"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API group with auth
	api := g.Group("/api", authMiddleware(app.store))
	{
		// User
		api.GET("/user/profile", app.getProfile)
		api.PUT("/user/profile", app.updateProfile)

		// Rooms
		api.GET("/rooms", app.listRooms)
		api.POST("/rooms", app.createRoom)
		api.GET("/rooms/:roomId", app.getRoom)
		api.POST("/rooms/:roomId/join", app.joinRoomHTTP)
		api.POST("/rooms/:roomId/leave", app.leaveRoomHTTP)
		api.GET("/rooms/:roomId/messages", app.getMessages)

		// Reactions
		api.PUT("/rooms/:roomId/messages/:messageId/reactions/:emoji", app.addReactionHTTP)
		api.DELETE("/rooms/:roomId/messages/:messageId/reactions/:emoji", app.removeReactionHTTP)

		// Seen
		api.PUT("/rooms/:roomId/messages/:messageId/seen", app.markSeenHTTP)

		// Threads
		api.GET("/rooms/:roomId/threads", app.listThreads)
		api.POST("/rooms/:roomId/threads", app.createThreadHTTP)
		api.GET("/threads/:threadId", app.getThread)
		api.GET("/threads/:threadId/messages", app.getThreadMessages)
	}

	// WebSocket (also protected by auth)
	g.GET("/ws", authMiddleware(app.store), app.wsHandler)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      g,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Chat MVP server (Gin) listening on http://localhost:8080 ...")
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ars1364/webrdp-gateway/backend/internal/core"
)

// connReq is the only shape accepted for create/update. Password nil on
// update means "keep the stored one"; "" means "clear it".
type connReq struct {
	Name       string  `json:"name"`
	Host       string  `json:"host"`
	Port       int     `json:"port"`
	Username   string  `json:"username"`
	Domain     string  `json:"domain"`
	Password   *string `json:"password"`
	Security   string  `json:"security"`
	IgnoreCert bool    `json:"ignore_cert"`
}

var securityModes = map[string]bool{"any": true, "nla": true, "tls": true, "rdp": true}

func (c *connReq) validate(needName bool) []fieldErr {
	var errs []fieldErr
	c.Name, c.Host = strings.TrimSpace(c.Name), strings.TrimSpace(c.Host)
	if needName && (c.Name == "" || len(c.Name) > 100) {
		errs = append(errs, fieldErr{"name", "1-100 characters"})
	}
	if c.Host == "" || len(c.Host) > 253 || strings.ContainsAny(c.Host, " /\\@?#") {
		errs = append(errs, fieldErr{"host", "an IP address or hostname"})
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fieldErr{"port", "1-65535"})
	}
	if len(c.Username) > 256 || len(c.Domain) > 256 {
		errs = append(errs, fieldErr{"username", "max 256 characters"})
	}
	if c.Password != nil && len(*c.Password) > 512 {
		errs = append(errs, fieldErr{"password", "max 512 characters"})
	}
	if c.Security == "" {
		c.Security = "any"
	}
	if !securityModes[c.Security] {
		errs = append(errs, fieldErr{"security", "one of any, nla, tls, rdp"})
	}
	return errs
}

func (s *Server) listConnections(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	p, ok := parsePage(w, r)
	if !ok {
		return
	}
	list, total, err := s.repo.ListConnections(r.Context(), sess.UserID, p)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeList(w, list, total, p)
}

func (s *Server) createConnection(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	var req connReq
	if !decode(w, r, &req) {
		return
	}
	if errs := req.validate(true); len(errs) > 0 {
		writeErr(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Check the highlighted fields.", errs...)
		return
	}
	c := req.toConn(newUUID())
	if err := s.sealPassword(c, req.Password); err != nil {
		s.internal(w, r, err)
		return
	}
	if err := s.repo.CreateConnection(r.Context(), sess.UserID, c); err != nil {
		s.internal(w, r, err)
		return
	}
	s.repo.Audit(r.Context(), sess.UserID, "connection.create", c.Name, clientIP(r), nil)
	writeJSON(w, http.StatusCreated, map[string]any{"data": c})
}

func (s *Server) updateConnection(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	var req connReq
	if !decode(w, r, &req) {
		return
	}
	if errs := req.validate(true); len(errs) > 0 {
		writeErr(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Check the highlighted fields.", errs...)
		return
	}
	if !isUUID(r.PathValue("id")) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Connection not found.")
		return
	}
	c := req.toConn(r.PathValue("id"))
	if err := s.sealPassword(c, req.Password); err != nil {
		s.internal(w, r, err)
		return
	}
	err := s.repo.UpdateConnection(r.Context(), sess.UserID, c, req.Password == nil)
	if errors.Is(err, core.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Connection not found.")
		return
	} else if err != nil {
		s.internal(w, r, err)
		return
	}
	s.repo.Audit(r.Context(), sess.UserID, "connection.update", c.Name, clientIP(r), nil)
	updated, err := s.repo.GetConnection(r.Context(), sess.UserID, c.ID)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": updated})
}

func (s *Server) deleteConnection(w http.ResponseWriter, r *http.Request, sess *core.Session) {
	id := r.PathValue("id")
	if !isUUID(id) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Connection not found.")
		return
	}
	err := s.repo.DeleteConnection(r.Context(), sess.UserID, id)
	if errors.Is(err, core.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "Connection not found.")
		return
	} else if err != nil {
		s.internal(w, r, err)
		return
	}
	s.repo.Audit(r.Context(), sess.UserID, "connection.delete", id, clientIP(r), nil)
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]bool{"ok": true}})
}

func (req *connReq) toConn(id string) *core.Connection {
	return &core.Connection{ID: id, Name: req.Name, Host: req.Host, Port: req.Port,
		Username: req.Username, Domain: req.Domain, Security: req.Security, IgnoreCert: req.IgnoreCert}
}

func (s *Server) sealPassword(c *core.Connection, pw *string) error {
	if pw == nil || *pw == "" {
		return nil
	}
	enc, err := s.sealer.Seal([]byte(*pw), []byte("conn:"+c.ID))
	c.PasswordEnc, c.HasPassword = enc, err == nil
	return err
}

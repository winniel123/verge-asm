package main

import (
	"context"
	"net/http"
	"strconv"

	"github.com/winniel123/verge-asm/internal/db"
)

type vergeCoreStore interface {
	DeleteVergeCoreFrequencyEdit(ctx context.Context, port int32) error
	UpsertVergeCoreFrequencyEdit(ctx context.Context, arg db.UpsertVergeCoreFrequencyEditParams) error
}

// The sensitive half is release-authored, so no write path here may reach it (v1-spec §3.5).

type freqRow struct {
	Port          int
	AlsoSensitive bool
	Edited        bool
	EditAction    string
}

type sensRow struct {
	Port      int
	Transport string
	Service   string
}

// Keyed on port alone because the one port on both transports, 11211, is memcached on each.

var sensitiveServiceLabels = map[int]string{
	21: "ftp", 23: "telnet", 69: "tftp", 137: "netbios-ns", 138: "netbios-dgm", 139: "netbios-ssn",
	445: "smb", 512: "rexec", 513: "rlogin", 514: "rsh", 623: "ipmi", 873: "rsync",
	2049: "nfs", 2181: "zookeeper", 2375: "docker", 2376: "docker-tls", 2379: "etcd", 2380: "etcd-peer",
	3306: "mysql", 4369: "epmd", 5432: "postgres", 5900: "vnc", 5984: "couchdb", 6000: "x11",
	6379: "redis", 9042: "cassandra", 10248: "kubelet-healthz", 10249: "kube-proxy-metrics",
	10250: "kubelet", 10255: "kubelet-ro", 10257: "kube-controller-manager", 10259: "kube-scheduler",
	11211: "memcached", 25672: "rabbitmq-dist", 27017: "mongodb", 27018: "mongodb-shard", 27019: "mongodb-config",
}

func (s *server) vergeCorePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A literal tab here would drift from the mapping failSettings stamps, dropping every callout.
	s.renderSettings(w, r, acct, s.takeSettingsFlash(r, tabForSection("vergecore")))
}

func (s *server) editVergeCoreFrequency(w http.ResponseWriter, r *http.Request, acct db.Account) {
	action := r.FormValue("action")
	portRaw := r.FormValue("port")
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{section: "vergecore", vcError: msg, vcPort: portRaw})
	}

	port, err := strconv.Atoi(portRaw)
	if err != nil || port < 1 || port > 65535 {
		fail("Enter a port between 1 and 65535.")
		return
	}

	switch action {
	case "add", "remove":
		if err := s.vergeCoreStore.UpsertVergeCoreFrequencyEdit(r.Context(), db.UpsertVergeCoreFrequencyEditParams{
			Port: int32(port), Action: action, CreatedBy: acct.ID, // #nosec G109 (port validated 1..65535 above)
		}); err != nil {
			s.serverError(w, "upsert verge-core frequency edit", err)
			return
		}
	case "reset":
		if err := s.vergeCoreStore.DeleteVergeCoreFrequencyEdit(r.Context(), int32(port)); err != nil { // #nosec G109 (port validated 1..65535 above)
			s.serverError(w, "delete verge-core frequency edit", err)
			return
		}
	default:
		fail("Choose add, remove or reset.")
		return
	}
	s.backToSection(w, r, "vergecore")
}

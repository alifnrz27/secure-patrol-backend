package service

import (
	"secure-patrol-backend/pkg/license"
	"testing"
	"time"
)

func TestEvaluate(t *testing.T) {
	t.Setenv("APP_MASTER_SECRET", "test-master-secret-0123456789abcdefghijk")
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	payload := &license.Payload{
		LicenseID: "LIC-1", InstallID: "SP-X", MaxUnits: 1, MaxAppClients: 1,
		IssuedAt: now.AddDate(-1, 0, 0), ExpiresAt: now.AddDate(0, 2, 0), GraceDays: 14,
	}

	cases := []struct {
		name     string
		v        verified
		at       time.Time
		want     State
		locked   bool
		expiring bool
	}{
		{"missing", verified{missing: true}, now, StateMissing, true, false},
		{"active, far from expiry", verified{payload: payload}, now, StateActive, false, false},
		{"active, expires within 30 days", verified{payload: payload}, payload.ExpiresAt.AddDate(0, 0, -10), StateActive, false, true},
		{"grace", verified{payload: payload}, payload.ExpiresAt.Add(time.Hour), StateGrace, false, true},
		{"expired after grace", verified{payload: payload}, payload.GraceUntil().Add(time.Second), StateExpired, true, false},
		{"invalid code", verified{invalid: "bad"}, now, StateInvalid, true, false},
		{"clock moved back", verified{payload: payload, clock: "behind"}, now, StateInvalid, true, false},
	}
	for _, c := range cases {
		status := evaluate(c.v, now, c.at)
		if status.State != c.want || status.Locked() != c.locked || (status.ExpiringIn != nil) != c.expiring {
			t.Errorf("%s: got state=%s locked=%v expiring=%v", c.name, status.State, status.Locked(), status.ExpiringIn != nil)
		}
	}
}

func TestUsageOver(t *testing.T) {
	if !(Usage{Units: 3, MaxUnits: 2}).UnitsOver() || (Usage{Units: 2, MaxUnits: 2}).UnitsOver() {
		t.Error("units over limit")
	}
	if (Usage{AppClients: 5}).AppClientsOver() {
		t.Error("no limit known must not be over")
	}
}

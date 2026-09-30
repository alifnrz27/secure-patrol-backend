package helper

import "github.com/gofiber/fiber/v2"

// LocalUnitID holds the *int64 unit of the logged in user (nil for central roles).
const LocalUnitID = "x-unit-id"

// Scope is the logged in user and the unit whose data they may access.
// Central users (UnitID nil) see every unit; unit users only their own.
type Scope struct {
	UserID   int64
	RoleCode string
	UnitID   *int64
}

func (s Scope) IsCentral() bool {
	return s.UnitID == nil
}

// UnitFilter returns the unit to filter on: always the user's own unit for a
// unit user (whatever was requested), and the requested unit (0 = all) for a
// central user.
func (s Scope) UnitFilter(requested int64) int64 {
	if s.UnitID != nil {
		return *s.UnitID
	}
	return requested
}

// CanAccessUnit reports whether the scope may see data of unitID.
func (s Scope) CanAccessUnit(unitID int64) bool {
	return s.UnitID == nil || *s.UnitID == unitID
}

func CurrentScope(c *fiber.Ctx) Scope {
	unitID, _ := c.Locals(LocalUnitID).(*int64)
	return Scope{UserID: CurrentUserID(c), RoleCode: CurrentRoleCode(c), UnitID: unitID}
}

// UnitFilterFromQuery applies UnitFilter to the ?unit_id query parameter.
func UnitFilterFromQuery(c *fiber.Ctx) int64 {
	requested := c.QueryInt("unit_id", 0)
	if requested < 0 {
		requested = 0
	}
	return CurrentScope(c).UnitFilter(int64(requested))
}

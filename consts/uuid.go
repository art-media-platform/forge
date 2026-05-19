package consts

import (
	"encoding/binary"
	"fmt"

	"github.com/gofrs/uuid/v5"
)

// rewriteUUIDLiterals normalizes UUID-string forms of `uid` constants
// into the canonical UIDLit{Hi, Lo} hex-pair form, so downstream emitters
// see a single representation.  Accepts every form gofrs/uuid.FromString
// accepts: canonical (dashes), compact (no dashes), brace-wrapped, and
// urn:uuid: prefixed.  Case-insensitive.
//
// Fails fast with file:line information if any uid string fails to parse.
func rewriteUUIDLiterals(cf *ConstFile) error {
	for _, decl := range cf.Decls {
		if decl.Const != nil {
			if err := rewriteOne(decl.Const); err != nil {
				return err
			}
		}
		if decl.ConstGrp != nil {
			for _, member := range decl.ConstGrp.Members {
				if err := rewriteOne(member); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func rewriteOne(cd *ConstDecl) error {
	if cd.Type != "uid" || cd.Value == nil || cd.Value.String == nil {
		return nil
	}

	literal := *cd.Value.String
	hi, lo, err := parseUUIDLiteral(literal)
	if err != nil {
		return fmt.Errorf("%s: const uid %s = %q: %w",
			cd.Pos, cd.VarName, literal, err)
	}

	cd.Value.UIDPair = &UIDLit{
		Pos: cd.Value.Pos,
		Hi:  fmt.Sprintf("0x%016X", hi),
		Lo:  fmt.Sprintf("0x%016X", lo),
	}
	cd.Value.String = nil
	return nil
}

// parseUUIDLiteral parses a UUID string into a big-endian uint64 pair.
// Matches the encoding amp.SDK previously exposed via UID_FromUUID.
func parseUUIDLiteral(s string) (hi, lo uint64, err error) {
	parsed, err := uuid.FromString(s)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid UUID literal: %w", err)
	}
	hi = binary.BigEndian.Uint64(parsed[0:8])
	lo = binary.BigEndian.Uint64(parsed[8:16])
	return hi, lo, nil
}

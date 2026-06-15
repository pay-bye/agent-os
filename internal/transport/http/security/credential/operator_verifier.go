package credential

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const operatorDigestAlgorithm = "sha256-base64url"

var (
	errMissingOperatorVerifier = errors.New("missing_operator_verifier")
	errInvalidOperatorVerifier = errors.New("invalid_operator_verifier")
)

type OperatorVerifierInput struct {
	File string
}

type OperatorKeyVerifier interface {
	Accepts(string) bool
}

type OperatorVerifier struct {
	digest [sha256.Size]byte
}

func (v OperatorVerifier) Accepts(key string) bool {
	sum := sha256.Sum256([]byte(key))
	return subtle.ConstantTimeCompare(sum[:], v.digest[:]) == 1
}

type operatorVerifierDocument struct {
	algorithm string
	digest    string
}

func LoadOperatorVerifier(input OperatorVerifierInput) (OperatorVerifier, error) {
	if input.File == "" {
		return OperatorVerifier{}, errMissingOperatorVerifier
	}
	digest, err := readOperatorVerifierDigest(input.File)
	if err != nil {
		return OperatorVerifier{}, errInvalidOperatorVerifier
	}
	return newOperatorVerifier(digest)
}

func readOperatorVerifierDigest(path string) (string, error) {
	root, name, err := openFileRoot(path)
	if err != nil {
		return "", err
	}
	defer root.Close()
	info, err := root.Stat(name)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		return "", errInvalidOperatorVerifier
	}
	content, err := root.ReadFile(name)
	if err != nil {
		return "", err
	}
	return decodeOperatorVerifierDigest(content)
}

func decodeOperatorVerifierDigest(content []byte) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	document, err := decodeOperatorVerifierDocument(decoder)
	if err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return "", errInvalidOperatorVerifier
	}
	if document.algorithm != operatorDigestAlgorithm || document.digest == "" {
		return "", errInvalidOperatorVerifier
	}
	return document.digest, nil
}

func newOperatorVerifier(value string) (OperatorVerifier, error) {
	digest, err := parseOperatorDigest(value)
	if err != nil {
		return OperatorVerifier{}, err
	}
	return OperatorVerifier{digest: digest}, nil
}

func parseOperatorDigest(value string) ([sha256.Size]byte, error) {
	var digest [sha256.Size]byte
	if value == "" || strings.TrimSpace(value) != value || strings.Contains(value, "=") {
		return digest, errInvalidOperatorVerifier
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return digest, errInvalidOperatorVerifier
	}
	canonical := base64.RawURLEncoding.EncodeToString(decoded)
	if canonical != value {
		return digest, errInvalidOperatorVerifier
	}
	copy(digest[:], decoded)
	return digest, nil
}

func decodeOperatorVerifierDocument(decoder *json.Decoder) (operatorVerifierDocument, error) {
	token, err := decoder.Token()
	if err != nil {
		return operatorVerifierDocument{}, errInvalidOperatorVerifier
	}
	if token != json.Delim('{') {
		return operatorVerifierDocument{}, errInvalidOperatorVerifier
	}
	document, err := readOperatorVerifierMembers(decoder)
	if err != nil {
		return operatorVerifierDocument{}, err
	}
	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') {
		return operatorVerifierDocument{}, errInvalidOperatorVerifier
	}
	return document, nil
}

func readOperatorVerifierMembers(decoder *json.Decoder) (operatorVerifierDocument, error) {
	var document operatorVerifierDocument
	seen := map[string]bool{}
	for decoder.More() {
		name, err := readOperatorVerifierMemberName(decoder, seen)
		if err != nil {
			return operatorVerifierDocument{}, err
		}
		value, err := readOperatorVerifierString(decoder)
		if err != nil {
			return operatorVerifierDocument{}, err
		}
		switch name {
		case "algorithm":
			document.algorithm = value
		case "digest":
			document.digest = value
		default:
			return operatorVerifierDocument{}, errInvalidOperatorVerifier
		}
	}
	return document, nil
}

func readOperatorVerifierMemberName(decoder *json.Decoder, seen map[string]bool) (string, error) {
	token, err := decoder.Token()
	if err != nil {
		return "", errInvalidOperatorVerifier
	}
	name, ok := token.(string)
	if !ok || seen[name] {
		return "", errInvalidOperatorVerifier
	}
	seen[name] = true
	return name, nil
}

func readOperatorVerifierString(decoder *json.Decoder) (string, error) {
	var value string
	if err := decoder.Decode(&value); err != nil {
		return "", errInvalidOperatorVerifier
	}
	return value, nil
}

func openFileRoot(path string) (*os.Root, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return nil, "", err
	}
	return root, filepath.Base(absolute), nil
}

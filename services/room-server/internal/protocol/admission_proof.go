// 进房准入凭证解析

package protocol

import "fmt"

// AdmissionProof is the identity and API-issued signature carried by the
// measured legacy JoinGame layout. Treat both as untrusted until verified.
type AdmissionProof struct {
	AccountID string
	Signature [32]byte
}

// 解析进房准入凭证
func DecodeAdmissionProof(frame []byte) (AdmissionProof, error) {
	_, proof, err := DecodeJoinAdmission(frame)
	return proof, err
}

// DecodeJoinAdmission decodes the join fingerprint and proof from the same frame.
func DecodeJoinAdmission(frame []byte) (JoinFingerprint, AdmissionProof, error) {
	join, err := DecodeJoinGameFingerprint(frame)
	if err != nil {
		return JoinFingerprint{}, AdmissionProof{}, err
	}
	if join.ServerVersion != 284 || join.ClientVersion != 542 {
		return JoinFingerprint{}, AdmissionProof{}, fmt.Errorf("unverified admission proof layout for %d/%d", join.ServerVersion, join.ClientVersion)
	}

	uuid := join.UUID
	proof := AdmissionProof{
		AccountID: fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]),
	}
	const signatureOffset = clientHeader + len(uuid)
	copy(proof.Signature[:], frame[signatureOffset:signatureOffset+len(proof.Signature)])
	return join, proof, nil
}

package keysharecore

import (
	"crypto/rand"
	"crypto/rsa"
	"sync"

	"github.com/privacybydesign/gabi/big"
	"github.com/privacybydesign/gabi/gabikeys"
	irma "github.com/privacybydesign/irmago"
)

type (
	AesKey [32]byte

	Core struct {
		// Keys used for storage encryption/decryption
		decryptionKeys  map[uint32]AesKey
		encryptionKey   AesKey
		encryptionKeyID uint32

		// Key used to sign keyshare protocol messages
		signKey   *rsa.PrivateKey
		signKeyID uint32

		// Commit values generated in first step of keyshare protocol
		commitmentData  map[uint64]*big.Int
		commitmentMutex sync.Mutex

		// IRMA issuer keys that are allowed to be used in keyshare
		//  sessions
		trustedKeys map[irma.PublicKeyIdentifier]*gabikeys.PublicKey
	}

	Configuration struct {
		AESKeyID  uint32
		AESKey    AesKey
		SignKeyID uint32
		SignKey   *rsa.PrivateKey
	}
)

func NewKeyshareCore(conf *Configuration) *Core {
	c := &Core{
		decryptionKeys: map[uint32]AesKey{},
		commitmentData: map[uint64]*big.Int{},
		trustedKeys:    map[irma.PublicKeyIdentifier]*gabikeys.PublicKey{},
	}
	c.setAESEncryptionKey(conf.AESKeyID, conf.AESKey)
	c.setSignKey(conf.SignKeyID, conf.SignKey)
	return c
}

func GenerateAESKey() (AesKey, error) {
	var res AesKey
	_, err := rand.Read(res[:])
	return res, err
}

// Add an aes key for decryption, with identifier keyid
// Calling this will cause all keyshare packets generated with the key to be trusted
func (c *Core) DangerousAddAESKey(keyID uint32, key AesKey) {
	c.decryptionKeys[keyID] = key
}

// Set the aes key for encrypting new/changed keyshare data
// with identifier keyid
// Calling this wil also cause all keyshare packets generated with the key to be trusted
func (c *Core) setAESEncryptionKey(keyID uint32, key AesKey) {
	c.decryptionKeys[keyID] = key
	c.encryptionKey = key
	c.encryptionKeyID = keyID
}

// Set key used to sign keyshare protocol messages
func (c *Core) setSignKey(id uint32, key *rsa.PrivateKey) {
	c.signKey = key
	c.signKeyID = id
}

// Add public key as trusted by keyshareCore. Calling this on incorrectly generated key material WILL compromise keyshare secrets!
func (c *Core) DangerousAddTrustedPublicKey(keyID irma.PublicKeyIdentifier, key *gabikeys.PublicKey) {
	c.trustedKeys[keyID] = key
}

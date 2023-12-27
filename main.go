package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"

	"github.com/joho/godotenv"
	"github.com/miekg/pkcs11"
)

const (
	defaultLibPath            = "/usr/local/Cellar/p11-kit/0.25.3/lib/pkcs11/yubihsm_pkcs11.dylib"
	defaultAuthKeyObjId       = "0002"
	defaultAuthKeyPassword    = "newpassword123"
	defaultAsymmetricKeyLabel = "hsm-go-test-key1"
	defaultDataFilePath       = "./data.txt"
)

func main() {
	// Load the .env file, and ignore the error if it doesn't exist
	_ = godotenv.Load()

	// Check if the environment variables are set.
	// If not, use the default values (from the const).
	libPath, exists := os.LookupEnv("PKCS11_LIB_PATH")
	if !exists {
		libPath = defaultLibPath
	}
	authKeyObjId, exists := os.LookupEnv("AUTH_KEY_OBJ_ID")
	if !exists {
		authKeyObjId = defaultAuthKeyObjId
	}
	authKeyPassword, exists := os.LookupEnv("AUTH_KEY_PASSWORD")
	if !exists {
		authKeyPassword = defaultAuthKeyPassword
	}
	asymmetricKeyLabel, exists := os.LookupEnv("ASYMMETRIC_KEY_LABEL")
	if !exists {
		asymmetricKeyLabel = defaultAsymmetricKeyLabel
	}
	dataFilePath, exists := os.LookupEnv("DATA_FILE_PATH")
	if !exists {
		dataFilePath = defaultDataFilePath
	}

	p := pkcs11.New(libPath)
	err := p.Initialize()
	if err != nil {
		panic(err)
	}
	defer p.Destroy()
	defer p.Finalize()

	slots, err := p.GetSlotList(true)
	if err != nil {
		panic(err)
	}

	session, err := p.OpenSession(slots[0], pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		panic(err)
	}
	defer p.CloseSession(session)

	err = p.Login(session, pkcs11.CKU_USER, authKeyObjId+authKeyPassword)
	if err != nil {
		panic(err)
	}
	defer p.Logout(session)

	// Find all objects
	err = p.FindObjectsInit(session, nil) // No template means "all objects"
	if err != nil {
		panic(err)
	}

	// Retrieve objects
	objects, _, err := p.FindObjects(session, 100)
	if err != nil {
		panic(err)
	}

	err = p.FindObjectsFinal(session)
	if err != nil {
		panic(err)
	}

	// List objects
	fmt.Printf("Found %d object(s)\n", len(objects))
	for _, obj := range objects {
		// Convert object handle to hex
		objIDHex := fmt.Sprintf("0x%016x", obj)

		// Get label attribute of the object
		labelAttr := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, nil),
		}
		labelAttrValue, err := p.GetAttributeValue(session, obj, labelAttr)
		if err != nil {
			fmt.Printf("Error getting label for object ID %s: %v\n", objIDHex, err)
			continue
		}

		// Get class attribute of the object
		classAttr := []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, nil),
		}
		classAttrValue, err := p.GetAttributeValue(session, obj, classAttr)
		if err != nil {
			fmt.Printf("Error getting class for object ID %s: %v\n", objIDHex, err)
			continue
		}

		// Read attributes
		label := ""
		if len(labelAttrValue[0].Value) > 0 {
			label = string(labelAttrValue[0].Value)
		}
		class := ""
		if len(classAttrValue[0].Value) > 0 {
			class = fmt.Sprintf("0x%x", classAttrValue[0].Value)
		}

		fmt.Printf("Object ID: %s, Label: %s, Class: %s\n", objIDHex, label, class)
	}

	// Find an asymmetric key private key object
	templatePrivKeyObj := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PRIVATE_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, asymmetricKeyLabel),
	}

	// Find private key object
	err = p.FindObjectsInit(session, templatePrivKeyObj)
	if err != nil {
		panic(err)
	}

	objectsPrivKey, _, err := p.FindObjects(session, 10)
	if err != nil {
		panic(err)
	}

	err = p.FindObjectsFinal(session)
	if err != nil {
		panic(err)
	}

	// Check if the private key is found
	if len(objectsPrivKey) == 0 {
		fmt.Println("Private key not found")
		return
	}
	privateKeyObject := objectsPrivKey[0]

	// Let's make sure that I can't read the private key value ;-)
	// Trying to get the private key value...
	getPrivKeyAttrTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, nil),
	}
	getPrivKeyAttrValue, err := p.GetAttributeValue(session, privateKeyObject, getPrivKeyAttrTemplate)
	if err != nil && len(getPrivKeyAttrValue) == 0 {
		fmt.Printf("Private Key is properly protected.\n")
	} else {
		panic("This should never happen. Private key value was leaked!")
	}

	// Find an asymmetric key (public key) object
	templatePubKeyObj := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, asymmetricKeyLabel),
	}
	err = p.FindObjectsInit(session, templatePubKeyObj)
	if err != nil {
		panic(err)
	}
	objectsPubKey, _, err := p.FindObjects(session, 10)
	if err != nil {
		panic(err)
	}
	err = p.FindObjectsFinal(session)
	if err != nil {
		panic(err)
	}

	// Check if the public key is found
	if len(objectsPubKey) == 0 {
		fmt.Println("Public key not found")
		return
	}
	publicKeyObject := objectsPubKey[0]

	// Convert object handle to hex
	publicKeyObjectIDHex := fmt.Sprintf("0x%016x", publicKeyObject)
	fmt.Printf("Using Public Key with Object ID: %s\n", publicKeyObjectIDHex)

	// Get the public key value
	getPubKeyAttrTemplate := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_VALUE, nil),
	}
	getPubKeyAttrValue, err := p.GetAttributeValue(session, publicKeyObject, getPubKeyAttrTemplate)
	if err != nil {
		panic(err)
	}

	publicKeyValue := getPubKeyAttrValue[0].Value
	publicKeyValueBase64 := base64.StdEncoding.EncodeToString(publicKeyValue)
	fmt.Printf("Public Key in Hex: %s\n", hex.EncodeToString(publicKeyValue))
	fmt.Printf("Public Key in Base64: %s\n", publicKeyValueBase64)

	// Read the data file to be signed
	data, err := os.ReadFile(dataFilePath)
	if err != nil {
		panic(err)
	}

	// Hash the data
	hashSignature := sha256.Sum256(data)

	// Request the HSM to sign the hash
	mechanism := []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_ECDSA, nil)}
	err = p.SignInit(session, mechanism, privateKeyObject)
	if err != nil {
		panic(err)
	}
	signature, err := p.Sign(session, hashSignature[:])
	if err != nil {
		panic(err)
	}
	fmt.Printf("Signature in Hex: %s\n", hex.EncodeToString(signature))
	fmt.Printf("Signature in Base64: %s\n", base64.StdEncoding.EncodeToString(signature))

	// Parse the public key
	pubKey, err := parseECPublicKeyFromBase64(publicKeyValueBase64)
	if err != nil {
		panic(err)
	}

	// Print the curve name based on the public key
	printCurveName(pubKey)

	// Prepare the signature for verification (extract r and s)
	r := big.NewInt(0).SetBytes(signature[:32])
	s := big.NewInt(0).SetBytes(signature[32:])

	// Verify the signature
	hashVerification := sha256.Sum256(data)
	verified := ecdsa.Verify(pubKey, hashVerification[:], r, s)
	if !verified {
		fmt.Println("Verification failed")
	} else {
		fmt.Println("Verification successful")
	}

}

func parseECPublicKeyFromBytes(pubBytes []byte) (*ecdsa.PublicKey, error) {
	pubKey, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return nil, err
	}

	switch pub := pubKey.(type) {
	case *ecdsa.PublicKey:
		return pub, nil
	default:
		return nil, fmt.Errorf("not an ECDSA public key")
	}
}

func parseECPublicKeyFromBase64(pubBase64 string) (*ecdsa.PublicKey, error) {
	pubBytes, err := base64.StdEncoding.DecodeString(pubBase64)
	if err != nil {
		return nil, err
	}

	return parseECPublicKeyFromBytes(pubBytes)
}

func printCurveName(pubKey *ecdsa.PublicKey) {
	switch pubKey.Curve {
	case elliptic.P224():
		fmt.Println("Curve: P-224 (secp224r1)")
	case elliptic.P256():
		fmt.Println("Curve: P-256 (secp256r1)")
	case elliptic.P384():
		fmt.Println("Curve: P-384 (secp384r1)")
	case elliptic.P521():
		fmt.Println("Curve: P-521 (secp521r1)")
	default:
		// Not sure if this works...
		switch pubKey.Curve.Params().Name {
		case "secp256k1":
			fmt.Println("Curve: secp256k1")
		default:
			fmt.Println("Curve: Unknown")
		}
	}
}

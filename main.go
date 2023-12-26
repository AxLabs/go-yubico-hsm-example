package main

import "github.com/miekg/pkcs11"

func main() {

	p := pkcs11.New("/usr/local/Cellar/p11-kit/0.25.3/lib/pkcs11/yubihsm_pkcs11.dylib")
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

	err = p.Login(session, pkcs11.CKU_USER, "0001password")
	if err != nil {
		panic(err)
	}
	defer p.Logout(session)

	template := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_PUBLIC_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, "password"),
	}
	if err = p.FindObjectsInit(session, template); err != nil {
		panic(err)
	}
	_, _, err = p.FindObjects(session, 1)
	if err != nil {
		panic(err)
	}
	if err = p.FindObjectsFinal(session); err != nil {
		panic(err)
	}
}

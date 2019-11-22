package dawg

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBadCreds(t *testing.T) {
	tok, err := gettoken("no", "and no")
	if err == nil {
		t.Error("expected an error")
	}
	if tok != nil {
		t.Error("expected nil token")
	}

	tok, err = gettoken("", "")
	if err == nil {
		t.Error("expected an error")
	}
	if tok != nil {
		t.Error("expected nil token")
	}

	tok, err = gettoken("5uup;hrg];ht8bijer$u9tot", "hurieahgr9[0249eingurivja")
	if err == nil {
		t.Error("wow i accidentlly cracked someone's password:", tok)
	}
}

func gettestcreds() (string, string, bool) {
	u, p := os.Getenv("DOMINOS_TEST_USER"), os.Getenv("DOMINOS_TEST_PASS")
	if len(u) == 0 || len(p) == 0 {
		return u, p, false
	}
	return u, p, true
}

var (
	testToken *token
	testAuth  *auth
	testUser  *UserProfile
)

func getTestToken(uname, pass string) (*token, error) {
	var err error
	if testToken == nil {
		testToken, err = gettoken(uname, pass)
	}
	return testToken, err
}

func getTestAuth(uname, pass string) (*auth, error) {
	var err error
	if testAuth == nil {
		testAuth, err = newauth(uname, pass)
	}
	return testAuth, err
}

func getTestUser(uname, pass string) (*UserProfile, error) {
	var err error
	if testUser == nil {
		testUser, err = SignIn(uname, pass)
	}
	return testUser, err
}

func TestToken(t *testing.T) {
	username, password, ok := gettestcreds()
	if !ok {
		t.Skip()
	}
	copyclient := orderClient
	orderClient = &client{
		host: orderHost,
		Client: &http.Client{
			Timeout:       10 * time.Second,
			CheckRedirect: noRedirects,
		},
	}

	tok, err := gettoken(username, password)
	if err != nil {
		t.Error(err)
	}
	if tok == nil {
		t.Fatal("nil token")
	}
	testToken = tok
	if len(tok.AccessToken) == 0 {
		t.Error("didnt get a auth token")
	}
	if !strings.HasPrefix(tok.authorization(), "Bearer ") {
		t.Error("bad auth format")
	}
	if tok.transport == nil {
		t.Error("token should have a transport")
	}
	if tok.Type != "Bearer" {
		t.Error("these tokens are usually bearer tokens")
	}
	orderClient = copyclient
}

func TestAuth(t *testing.T) {
	username, password, ok := gettestcreds()
	if !ok {
		t.Skip()
	}

	auth, err := newauth(username, password)
	if err != nil {
		t.Error(err)
	}
	if auth == nil {
		t.Fatal("got nil auth")
	}
	testAuth = auth // save this for later
	if auth.token == nil {
		t.Fatal("needs token")
	}
	if len(auth.username) == 0 {
		t.Error("didnt save username")
	}
	if len(auth.password) == 0 {
		t.Error("didnt save password")
	}
	if auth.cli == nil {
		t.Fatal("needs to have client")
	}
	if len(auth.token.AccessToken) == 0 {
		t.Error("no access token")
	}

	user, err := auth.login()
	if err != nil {
		t.Error(err)
	}
	if user == nil {
		t.Fatal("got nil user-profile")
	}
	user.AddAddress(testAddress())
	user.Addresses[0].StreetNumber = ""
	user.Addresses[0].StreetName = ""
	user.AddAddress(user.Addresses[0])
	a1 := user.Addresses[0]
	a2 := user.Addresses[1]
	if a1.StreetName != a2.StreetName {
		t.Error("did not copy address name correctly")
	}
	if a1.StreetNumber != a2.StreetNumber {
		t.Error("did not copy address number correctly")
	}
	a1.Street = ""
	if user.Addresses[0].LineOne() != a2.LineOne() {
		t.Error("line one for UserAddress is broken")
	}

	store, err := user.NearestStore("Delivery")
	if err != nil {
		t.Error(err)
	}
	if store.cli == nil {
		t.Fatal("store did not get a client")
	}
	if store.cli.host != "order.dominos.com" {
		t.Error("store client has the wrong host")
	}
	req := &http.Request{
		Method: "GET", Host: store.cli.host, Proto: "HTTP/1.1",
		URL: &url.URL{
			Scheme: "https", Host: store.cli.host,
			Path:     fmt.Sprintf("/power/store/%s/menu", store.ID),
			RawQuery: (&Params{"lang": DefaultLang, "structured": "true"}).Encode()},
	}
	res, err := store.cli.Do(req)
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()
	authhead := res.Request.Header.Get("Authorization")
	if authhead != auth.token.authorization() {
		t.Error("store client didnt get the token")
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if len(b) == 0 {
		t.Error("zero length response")
	}
	menu, err := store.Menu()
	if err != nil {
		t.Error(err)
	}
	if menu == nil {
		t.Error("got nil menu")
	}
	o := store.NewOrder()
	if o == nil {
		t.Error("nil order")
	}
	_, err = o.Price()
	if err != nil {
		t.Error(err)
	}
}

func TestAuth_Err(t *testing.T) {
	a, err := newauth("not a", "valid password")
	if err == nil {
		t.Error("expected an error")
	}
	if a != nil {
		t.Error("expected a nil auth")
	}
	a = &auth{
		username: "not a",
		password: "valid password",
		token:    &token{}, // assume we alread have a token
		cli: &client{
			host: "order.dominos.com",
			Client: &http.Client{
				Timeout:       60 * time.Second,
				CheckRedirect: noRedirects,
			},
		},
	}

	user, err := a.login()
	if err == nil {
		t.Error("expected an error")
	}
	if user != nil {
		t.Errorf("expected a nil user: %+v", user)
	}
	a.cli.host = "invalid_host.com"
	user, err = a.login()
	if err == nil {
		t.Error("expedted an error")
	}
	if user != nil {
		t.Error("user should still be nil")
	}
}

func TestSignIn(t *testing.T) {
	username, password, ok := gettestcreds()
	if !ok {
		t.Skip()
	}

	user, err := SignIn(username, password)
	if err != nil {
		t.Error(err)
	}
	if user == nil {
		t.Fatal("got nil user from SignIn")
	}
	testUser = user
}

func TestAuthClient(t *testing.T) {
	username, password, ok := gettestcreds()
	if !ok {
		t.Skip()
	}

	auth, err := getTestAuth(username, password)
	if auth == nil {
		t.Fatal("got nil auth")
	}

	if auth.cli == nil {
		t.Error("client should not be nil")
	}
	err = auth.cli.CheckRedirect(nil, nil)
	if err == nil {
		t.Error("order Client should not allow redirects")
	}
	err = auth.cli.CheckRedirect(&http.Request{}, []*http.Request{})
	if err == nil {
		t.Error("expected error")
	}
	tok, err := gettoken("bad", "creds")
	if err == nil {
		t.Error("should return error")
	}

	req := newAuthRequest(oauthURL, url.Values{})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	tok = &token{}
	err = unmarshalToken(resp.Body, tok)
	if err == nil {
		t.Error("expected error")
	}
	if e, ok := err.(*tokenError); !ok {
		t.Error("expected a *tokenError as the error")
	} else if e.Error() != fmt.Sprintf("%s: %s", e.Err, e.ErrorDesc) {
		t.Error("wrong error message")
	}
	if IsOk(err) {
		t.Error("this shouldnt happen")
	}
}

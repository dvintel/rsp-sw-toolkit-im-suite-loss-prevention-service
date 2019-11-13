package notification

//func TestRegisterSubscriber(t *testing.T) {
//
//	httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
//
//		if request.Method != "POST" {
//			t.Errorf("Expected 'POST' request, received '%s", request.Method)
//		}
//		writer.WriteHeader(http.StatusCreated)
//
//	}))
//
//	if err := RegisterSubscriber([]string{""}); err != nil {
//		t.Error(err)
//	}
//
//}

//func TestPostNotification(t *testing.T) {
//
//	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
//
//		if request.Method != "POST" {
//			t.Errorf("Expected 'POST' request, received '%s", request.Method)
//		}
//		writer.WriteHeader(http.StatusAccepted)
//
//	}))
//
//	if err := PostNotification("", testServer.URL); err != nil {
//		t.Error(err)
//	}
//
//}

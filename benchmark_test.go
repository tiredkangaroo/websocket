package websocket_test

import (
	"net/http"
	"testing"
	"websocket"
	// coder "github.com/coder/websocket"
)

var conn = new(MockResponseWriterHijack)
var coderConn = new(MockResponseWriterHijack)

var wsconn *websocket.Conn

// var coderWSConn *coder.Conn

var loremIpsum = ` Lorem ipsum dolor sit amet, consectetur adipiscing elit. Mauris vestibulum a justo sit amet rhoncus. Aliquam laoreet dui at magna eleifend dapibus. In eget tincidunt diam. Vivamus id accumsan nulla. Nam eu dictum diam. Donec vel turpis mi. Cras blandit vestibulum dui. Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere cubilia curae; Phasellus commodo vehicula pulvinar. Nulla maximus tempus magna finibus egestas. Duis nec eros arcu. Etiam iaculis iaculis rhoncus.

Praesent in mollis elit. Vestibulum cursus consectetur nulla, quis hendrerit lectus. Orci varius natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Proin mattis, risus ut hendrerit rhoncus, ante diam sagittis lorem, id consequat ante tortor scelerisque lacus. Etiam non convallis eros. Sed malesuada eu neque vitae ullamcorper. Nulla nulla enim, varius eget gravida et, malesuada ultricies mauris. Integer malesuada lobortis turpis sit amet tristique. Integer tincidunt tellus nulla, quis vestibulum nisi volutpat sit amet. Proin sit amet ante at lectus hendrerit feugiat.

Etiam vitae nunc eu ex efficitur elementum. Phasellus vitae elit sed nunc facilisis elementum. Mauris ultricies, dui et fringilla molestie, arcu est feugiat tellus, nec suscipit nisi erat eget nunc. Integer id vulputate mauris, ut lacinia nulla. Suspendisse egestas lobortis est, et porttitor mauris tincidunt at. In iaculis est in tellus euismod euismod. Fusce elementum nisi a augue aliquet tristique. Quisque nec rhoncus augue.

Phasellus auctor convallis turpis a fringilla. Sed sollicitudin fringilla tempor. In accumsan ipsum ac commodo accumsan. Pellentesque sed purus at felis facilisis vehicula id nec urna. Sed lectus eros, tempor nec finibus tristique, feugiat eu libero. Integer rutrum placerat lorem, non mollis risus tincidunt at. Pellentesque sit amet aliquam nisl. Nam quis leo ut orci imperdiet pellentesque id sed lectus. Vivamus lacinia gravida odio, vestibulum auctor est consequat eget. Nullam a ullamcorper massa. Nulla facilisi. Integer lobortis aliquet urna, ut congue sem suscipit non. Sed convallis porttitor porttitor. Proin pellentesque ipsum eget sem rutrum condimentum. Maecenas a congue dolor. Suspendisse egestas viverra sem, sed consectetur ante molestie non. `
var req, _ = http.NewRequest("GET", "http://localhost/ws", nil)

func init() {
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==") // base64-encoded test key
}

func BenchmarkAccept(b *testing.B) {
	var err error
	wsconn, err = websocket.AcceptHTTP(conn, req)
	if err != nil {
		b.Fatal(err.Error())
	}
	b.ReportAllocs()
}

// func BenchmarkCoderAccept(b *testing.B) {
// 	var err error
// 	coderWSConn, err = coder.Accept(coderConn, req, nil)
// 	if err != nil {
// 		b.Fatal(err.Error())
// 	}
// 	b.ReportAllocs()
// }

func BenchmarkWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := wsconn.Write(&websocket.Message{
			Type: websocket.MessageText,
			Data: []byte("hello world"),
		})
		if err != nil {
			b.Fatalf(err.Error())
		}
	}
}

// func BenchmarkCoderWrite(b *testing.B) {
// 	ctx := context.Background()
// 	for i := 0; i < b.N; i++ {
// 		err := coderWSConn.Write(ctx, coder.MessageText, []byte("hello world"))
// 		if err != nil {
// 			b.Fatalf(err.Error())
// 		}
// 	}
// }

func BenchmarkWriteLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := wsconn.Write(&websocket.Message{
			Type: websocket.MessageText,
			Data: []byte(loremIpsum),
		})
		if err != nil {
			b.Fatalf(err.Error())
		}
	}
}

// func BenchmarkCoderWriteLarge(b *testing.B) {
// 	ctx := context.Background()
// 	for i := 0; i < b.N; i++ {
// 		err := coderWSConn.Write(ctx, coder.MessageText, []byte(loremIpsum))
// 		if err != nil {
// 			b.Fatalf(err.Error())
// 		}
// 	}
// }

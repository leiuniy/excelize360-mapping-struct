---
title: Golang中Wesocket的使用
date: 2022-03-01 22:04:51
img: https://gitee.com/chenlei888/picturehub/raw/master/img/202203012212285.png
tags:
    - golang
    - websocket
categories: Golang
keywords: golang,websocket
git: git@gitee.com:chenlei888/md-blog.git

---



# Golang中Wesocket的使用

[TOC]

## 一、什么是WebSockets

通常 Web 应用使用一个或多个请求对 HTTP 服务器提供对外服务。客户端软件通常是 Web 浏览器向服务器发送请求，服务器发回一个响应。响应通常是 HTML 内容，由浏览器来渲染为页面。样式表，JavaScript 代码和图像也可以在响应中发送回来以完成整个网页。每个请求和响应都属于特定的单独的连接的一部分，像 Facebook 这样的大型网站为了渲染单个页面实际上可以产生数百个这样的连接。

AJAX 的工作方式跟这个完全相同。使用 JavaScript，开发人员可以向 HTTP 服务器请求一小段信息，然后根据响应更新部分页面。这可以在不刷新浏览器的情况下完成，但仍然存在一些限制。

每个 HTTP 请求/响应的连接在被响应之后都会关闭，因此获得任何新的信息必须新建另一个连接。如果没有新的请求发送给服务器，它就不知道客户端正在查找新的信息。能让 AJAX 应用程序看起来像实时的一种技术是定时循环发送 AJAX 请求。在设置了时间间隔之后，应用程序可以重新将请求发送到服务器，以查看是否有任何更新需要反馈给浏览器。这比较适合小型应用程序，但并不高效。这时候 WebSockets 就派上用场了。

WebSockets 是由 Internet 工程任务组（IETF）创建的建议标准的一部分。 [RFC6455](https://tools.ietf.org/html/rfc6455) 中详细描述了 WebSockets 实现的完整技术规范。下面是该文档定义 WebSocket 的节选：

> WebSocket 协议用于客户端代码和远程主机之间进行通信，其中客户端代码是在可控环境下的非授信代码

换句话说，WebSocket 是一个总是打开的连接，允许客户端和服务器自发地来回发送消息。服务器可在必要时将新信息推送到客户端，客户端也可以对服务器执行相同操作。

## 二、第三方包实现

### 1、Go 中的 WebSockets

WebSockets 并不包含在 Go 标准库中，但幸运的是有一些不错的第三方包让 WebSockets 的使用轻而易举。在这个例子中，我们将使用一个名为“gorilla/websocket”的包，它是流行的 [Gorilla Toolkit](http://www.gorillatoolkit.org/) 包集合的一部分，多用于在 Go 中创建 Web 应用程序。请运行以下命令进行安装：

```
$ go get github.com/gorilla/websocket
```

### 2、JavaScript 中的 WebSockets

大多数现代浏览器都在其 JavaScript 实现中支持 WebSockets。要从浏览器中启动一个 WebSocket 连接，你可以使用简单的 WebSocket JavaScript 对象，如下：

```
var ws = new Websocket("ws://example.com/ws");
```

您唯一需要的参数是一个 URL，WebSocket 连接可通过此 URL 连接服务器。该请求实际是一个 HTTP 请求，但为了安全连接我们使用“ws://”或“wss://”。这使服务器知道我们正在尝试创建一个新的 WebSocket 连接。之后服务器将“升级”该客户端和服务之间的连接到永久的双向连接。

一旦新的 WebSocket 对象被创建，并且连接成功创建之后，我们就可以使用“send()”方法发送文本到服务器，并在 WebSocket 的“onmessage”属性上定义一个处理函数来处理从服务器发送的消息。具体逻辑会在之后的聊天应用程序代码中解释。

### 3、如何使用WebSockets

#### (1)构建服务器

这个应用程序的第一部分是服务器。这是一个处理请求的简单 HTTP 服务器。它将为我们提供 HTML5 和 JavaScript 代码，以及建立客户端的 WebSocket 连接。另外，服务器还将跟踪每个 WebSocket 连接并通过 WebSocket 连接将聊天信息从一个客户端发送到所有其他客户端。首先创建一个新的空目录，然后在该目录中创建一个“src”和“public”目录。在“src”目录中创建一个名为“main.go”的文件。

搭建服务器首先要进行一些设置。我们像所有 Go 应用程序一样启动应用程序，并定义包命名空间，在本例中为“main”。接下来我们导入一些有用的包。 “log”和“net/http”都是标准库的一部分，将用于日志记录并创建一个简单的 HTTP 服务器。最终包“github.com/gorilla/websocket”将帮助我们轻松创建和使用 WebSocket 连接。

```go
package main
import (
        "log"
        "net/http"

        "github.com/gorilla/websocket")
```

下面的两行代码是一些全局变量，在应用程序的其它地方会被用到。全局变量的实践较差，不过这次为了简单起见我们还是使用了它们。第一个变量是一个 map 映射，其键对应是一个指向 WebSocket 的指针，其值就是一个布尔值。我们实际上并不需要这个值，但使用的映射数据结构需要有一个映射值，这样做更容易添加和删除单项。

第二个变量是一个用于由客户端发送消息的队列，扮演通道的角色。在后面的代码中，我们会定义一个 goroutine 来从这个通道读取新消息，然后将它们发送给其它连接到服务器的客户端。

```go
var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan Message)           // broadcast channel
```

接下来我们创建一个 upgrader 的实例。这只是一个对象，它具备一些方法，这些方法可以获取一个普通 HTTP 链接然后将其升级成一个 WebSocket，稍后会有相关代码介绍。

```go
// Configure the upgrader
var upgrader = websocket.Upgrader{}
```

最后我们将定义一个对象来管理消息，数据结构比较简单，带有一些字符串属性，一个 email 地址，一个用户名以及实际的消息内容。我们将利用 email 来展示 [Gravatar](http://en.gravatar.com/) 服务所提供的唯一身份标识。

由反引号包含的文本是 Go 在对象和 JSON 之间进行序列化和反序列化时需要的元数据。

```go
// Define our message object
type Message struct {
        Email    string `json:"email"`
        Username string `json:"username"`
        Message  string `json:"message"`
        }
```

Go 应用程序的主要入口总是 "main()" 函数。代码非常简洁。我们首先创建一个静态的文件服务，并将之与 "/" 路由绑定，这样用户访问网站时就能看到 index.html 和其它资源。在这个示例中我们有一个保存 JavaScript 代码的 "app.js" 文件和一个保存样式的 "style.css" 文件。

```go
func main() {
        // Create a simple file server
        fs := http.FileServer(http.Dir("../public"))
        http.Handle("/", fs)
```

我们想定义的下一个路由是 "/ws"，在这里处理启动 WebSocket 的请求。我们先向处理函数传递一个函数的名称，"handleConnections"，稍后再来定义这个函数。

```go
func main() {
    ...
        // Configure websocket route
        http.HandleFunc("/ws", handleConnections)
```

下一步就是启动一个叫 "handleMessages" 的 Go 程序。这是一个并行过程，独立于应用和其它部分运行，从广播频道中取得消息并通过各客户端的 WebSocket 连接传递出去。并行是 Go 中一项强大的特性。关于它如何工作的内容超出了这篇文章的范围，不过你可以自行查看 Go 的[官方教程](https://tour.golang.org/concurrency/1)网站。如果你熟悉 JavaScript，可联想一下并行过程，作为后台过程运行的 Go 程序，或 JavaScript 的异步函数。

```go
func main() {
    ...
        // Start listening for incoming chat messages
        go handleMessages()
```

最后，我们向控制台打印一个辅助信息并启动 Web 服务。如果有错误发生，我们就把它记录下来然后退出应用程序。

```go
func main() {
    ...
        // Start the server on localhost port 8000 and log any errors
        log.Println("http server started on :8000")
        err := http.ListenAndServe(":8000", nil)
        if err != nil {
                log.Fatal("ListenAndServe: ", err)
        }
}
```

接下来我们创建一个函数处理传入的 WebSocket 连接。首先我们使用升级的 "Upgrade()" 方法改变初始的 GET 请求，使之成为完全的 WebSocket。如果发生错误，记录下来，但不退出。同时注意 defer 语句，它通知 Go 在函数返回的时候关闭 WebSocket。这是个不错的方法，它为我们节省了不少可能出现在不同分支中返回函数前的 "Close()" 语句。

```go
func handleConnections(w http.ResponseWriter, r *http.Request) {
        // Upgrade initial GET request to a websocket
        ws, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
                log.Fatal(err)
        }
        // Make sure we close the connection when the function returns
        defer ws.Close()
```

接下来把新的客户端添加到全局的 "clients" 映射表中进行注册，这个映射表在早先已经创建了。

```go
func handleConnections(w http.ResponseWriter, r *http.Request) {
    ...
        // Register our new client
        clients[ws] = true
```

最后一步是一个无限循环，它一直等待着要写入 WebSocket 的新消息，将其从 JSON 反序列化为 Message 对象然后送入广播频道。然而 "handleMessages()" Go 程序就能把它送给连接中的其它客户端。

如果从 socket 中读取数据有误，我们假设客户端已经因为某种原因断开。我们记录错误并从全局的 “clients” 映射表里删除该客户端，这样一来，我们不会继续尝试与其通信。

另外，HTTP 路由处理函数已经被作为 goroutines 运行。这使得 HTTP 服务器无需等待另一个连接完成，就能处理多个传入连接。

```go
func handleConnections(w http.ResponseWriter, r *http.Request) {
    ...
        for {
                var msg Message                // Read in a new message as JSON and map it to a Message object
                err := ws.ReadJSON(&msg)
                if err != nil {
                        log.Printf("error: %v", err)
                        delete(clients, ws)
                        break
                }
                // Send the newly received message to the broadcast channel
                broadcast <- msg        }
}
```

服务器的最后一部分是"handleMessages()"函数。这是一个简单循环，从“broadcast”中连续读取数据，然后通过各自的 WebSocket 连接将消息传播到所以客户端。同样，如果写入 Websocket 时出现错误，我们将关闭连接，并将其从“clients” 映射中删除。

```go
func handleMessages() {
        for {
                // Grab the next message from the broadcast channel
                msg := <-broadcast
                // Send it out to every client that is currently connected
                for client := range clients {
                        err := client.WriteJSON(msg)
                        if err != nil {
                                log.Printf("error: %v", err)
                                client.Close()
                                delete(clients, client)
                        }
                }
        }
}
```

#### (2)构建客户端

如果没有漂亮的 UI，聊天应用程序将无法完成。 我们需要使用一些 HTML5 和 VueJS 来创建一个简单、干净的界面，再利用一些诸如 [Materialize CSS](http://materializecss.com/) 和 [EmojiOn](http://emojione.com/) 的库来生成一些样式和表情符号。 在“public”目录中，创建一个名为“index.html”的新文件。

第一部分很基础。为了美观，我们也会放入一些样式表和字体。“style.css”是自定义的样式表，用于自定义一些内容。

```html
<!DOCTYPE html><html lang="en"><head>
    <meta charset="UTF-8">
    <title>Simple Chat</title>

    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/materialize/0.97.8/css/materialize.min.css">
    <link rel="stylesheet" href="https://fonts.googleapis.com/icon?family=Material+Icons">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/emojione/2.2.6/assets/css/emojione.min.css"/>
    <link rel="stylesheet" href="/style.css">
</head>
```

下一部分仅与接口相关，其中只包含一些用于选择用户名、发送消息和显示新的聊天信息的字段。与 VueJS 交互的细节超出本文的介绍范围，你可阅读[此文档](https://vuejs.org/v2/guide/#Handling-User-Input)了解更多。

```html
<body>
<header>
    <nav>
        <div class="nav-wrapper">
            <a href="/" class="brand-logo right">Simple Chat</a>
        </div>
    </nav>
</header>
<main id="app">
    <div class="row">
        <div class="col s12">
            <div class="card horizontal">
                <div id="chat-messages" class="card-content" v-html="chatContent">
                </div>
            </div>
        </div>
    </div>
    <div class="row" v-if="joined">
        <div class="input-field col s8">
            <input type="text" v-model="newMsg" @keyup.enter="send">
        </div>
        <div class="input-field col s4">
            <button class="waves-effect waves-light btn" @click="send">
                <i class="material-icons right">chat</i>
                Send
            </button>
        </div>
    </div>
    <div class="row" v-if="!joined">
        <div class="input-field col s8">
            <input type="email" v-model.trim="email" placeholder="Email">
        </div>
        <div class="input-field col s8">
            <input type="text" v-model.trim="username" placeholder="Username">
        </div>
        <div class="input-field col s4">
            <button class="waves-effect waves-light btn" @click="join()">
                <i class="material-icons right">done</i>
                Join
            </button>
        </div>
    </div>
</main>
<footer class="page-footer">
</footer>
```

最后一步只需要导入所有需要的 JavaScript 库，包括 Vue、EmojiOne、jQuery 和 Materialize。我们需要 MD5 库获取来自 [Gravatar](http://en.gravatar.com/) 的头像 URL，这用 JavaScript 代码写出来就好理解了。最后导入 "app.js"。

```html
<script src="https://unpkg.com/vue@2.1.3/dist/vue.min.js"></script>
<script src="https://cdn.jsdelivr.net/emojione/2.2.6/lib/js/emojione.min.js"></script>
<script src="https://code.jquery.com/jquery-2.1.1.min.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/crypto-js/3.1.2/rollups/md5.js"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/materialize/0.97.8/js/materialize.min.js"></script>
<script src="/app.js"></script>
</body>
</html>
```

然后在 "public" 目录下创建一个 "style.css" 文件。其中会放入一些样式。

```css
body {
    display: flex;
    min-height: 100vh;
    flex-direction: column;
}

main {
    flex: 1 0 auto;
}

#chat-messages {
    min-height: 10vh;
    height: 60vh;
    width: 100%;
    overflow-y: scroll;
}
```

客户端的最后一部分是 JavaScript 代码。在 "public" 目录下创建文件 "app.js"。

对于 VueJS 应用程序来说，一开始都是创建新的 Vue 对象。我们将它与 id 为 "#app" 的 div 绑定。这会让 div 内的所有东西与 Vue 实现共享作用域。下面定义一些变量。 

```javascript
new Vue({
    el: '#app',

    data: {
        ws: null, // Our websocket
        newMsg: '', // Holds new messages to be sent to the server
        chatContent: '', // A running list of chat messages displayed on the screen
        email: null, // Email address used for grabbing an avatar
        username: null, // Our username
        joined: false // True if email and username have been filled in
    },
```

Vue 提供了一个叫 "created" 的属性，这是一个函数，会在 Vue 实例刚刚创建时调用。这里非常适合对应用做一些设置工作。在这个示例中我们希望创建一个新的 WebSocket 连接与服务器连接，并创建一个处理器用于处理从服务器发送过来的消息。我们把新的 WebSocket 对象保存在 "data" 属性的 "ws" 变量中。 

"addEventListener()"方法接受一个用于处理传入消息的函数。我们期望所有消息都是 JSON 字符串，以便统一解析为一个对象字面量。然后我们可以用各个属性和 avater 头像一起组成漂亮的消息行。"gravatarURL()" 方法会在后面详述。我们用了一个叫 EmojiOne 的表情库来解析[emoji 代码](http://emoji.codes/)。"toImage()" 方法会把 emoji 代码转换为实际的图片。比如，如果你输入 ":robot:"，它会被替换为一个机器人 emoji 表情图。

```javascript
created: function() {
    var self = this;
    this.ws = new WebSocket('ws://' + window.location.host + '/ws');
    this.ws.addEventListener('message', function(e) {
        var msg = JSON.parse(e.data);
        self.chatContent += '<div class="chip">'
                + '<img src="' + self.gravatarURL(msg.email) + '">' // Avatar
                + msg.username
            + '</div>'
            + emojione.toImage(msg.message) + '<br/>'; // Parse emojis

        var element = document.getElementById('chat-messages');
        element.scrollTop = element.scrollHeight; // Auto scroll to the bottom
    });
},
```

"methods" 属性可以定义各种函数，我们会在 VueJS 应用中使用这些函数。"send"方法用于向服务器发送消息。我们先确保消息不是空的，然后把消息组织成一个对象，再用"stringify"把它变成 JSON 字符串，以便服务器能正确解析。我们使用 jQuery 来处理传入消息中 HTML 和 JavaScript 中的特殊字符，以防止各种类型的注入攻击。

```javascript
methods: {
    send: function () {
        if (this.newMsg != '') {
            this.ws.send(
                JSON.stringify({
                    email: this.email,
                    username: this.username,
                    message: $('<p>').html(this.newMsg).text() // Strip out html
                }
            ));
            this.newMsg = ''; // Reset newMsg
        }
    },
```

"join"函数会确保用户在发送消息前输入 email 地址和用户名。一旦他们输入了这些信息，我们将 joined 设置为 "true"，同时允许他们开始交谈。同样，我们会处理 HTML 和 JavaScript 的特殊字符。

```javascript
join: function () {
        if (!this.email) {
            Materialize.toast('You must enter an email', 2000);
            return
            }
        if (!this.username) {
            Materialize.toast('You must choose a username', 2000);
            return
        }
        this.email = $('<p>').html(this.email).text();
        this.username = $('<p>').html(this.username).text();
        this.joined = true;
    },
```

最后一个函数是一个很好的辅助函数，用于从 Gravatar 获取头像。URL 的最后一段需要用户的 email 地址的 MD5 编码。MD5 是一种加密算法，它能隐藏 email 地址同时还能让 email 地址作为一个唯一标识来使用。

```javascript
        gravatarURL: function(email) {
            return 'http://www.gravatar.com/avatar/' + CryptoJS.MD5(email);
        }
    }
});
```

### 4、运行应用程序

要运行该应用程序，请打开控制台窗口并确保进入应用程序的“src”目录中，然后运行以下命令。

```
$ go run main.go
```

![img](https://gitee.com/chenlei888/picturehub/raw/master/img/202203012212285.png)

接下来打开 Web 浏览器并导航到“[http://localhost:8000](http://localhost:8000/)”站点。 然后就会显示聊天屏幕，你可以在聊天屏幕中输入电子邮件和用户名。

![img](https://gitee.com/chenlei888/picturehub/raw/master/img/202203012212792.png)

如果要查看该应用多个用户之间的通信方式，只需另外打开一个浏览器标签页或窗口，然后导航到“[http://localhost:8000](http://localhost:8000/)”。 输入不同的电子邮件和用户名。然后轮流从两个窗口发送消息，这样就可以看到多个用户之间的通信方式了。

![img](https://gitee.com/chenlei888/picturehub/raw/master/img/202203012212729.png)

### 5、结论 

这只是一个基本的聊天应用程序，可以在此基础上进行更多的改进，添加更多的其他功能！

## 三、golang官方包实现

### 1、安装websocket

使用的golang官方的net包下面的websocket，地址：
https://github.com/golang/net

下载net包，安装websocket模块

```shell
#全部模块下载
go get github.com/golang/net
#做下软连接把github文件夹下面的映射到golang.org下，否则其他模块如html安装不上。
ln -s github.com/golang/net golang.org/x/net
#安装websocket模块
go install golang.org/x/net/websocket
```


这个模块的包结构都统一成golang.org/x/net。使用import “golang.org/x/net/websocket”引入。

官方文档:
https://godoc.org/golang.org/x/net/websocket

### 2、代码和运行

server代码：最终还是挂在http服务器上面的。

```go
package main

import (
    "golang.org/x/net/websocket"
    "fmt"
    "log"
    "net/http"
)

func echoHandler(ws *websocket.Conn) {
    msg := make([]byte, 512)
    n, err := ws.Read(msg)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Receive: %s\n", msg[:n])

    send_msg := "[" + string(msg[:n]) + "]"
    m, err := ws.Write([]byte(send_msg))
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Send: %s\n", msg[:m])
}

func main() {
    http.Handle("/echo", websocket.Handler(echoHandler))
    http.Handle("/", http.FileServer(http.Dir(".")))

    err := http.ListenAndServe(":8080", nil)

    if err != nil {
        panic("ListenAndServe: " + err.Error())
    }
}
```


客户端websocket调用代码：

```go
package main

import (
    "golang.org/x/net/websocket"
    "fmt"
    "log"
)

var origin = "http://127.0.0.1:8080/"
var url = "ws://127.0.0.1:8080/echo"

func main() {
    ws, err := websocket.Dial(url, "", origin)
    if err != nil {
        log.Fatal(err)
    }
    message := []byte("hello, world!你好")
    _, err = ws.Write(message)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Send: %s\n", message)

    var msg = make([]byte, 512)
    m, err := ws.Read(msg)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Receive: %s\n", msg[:m])

    ws.Close()//关闭连接

}
```

客户端使用websocket.Dial(url, “”, origin) 进行websocket连接，但是origin参数并没有实际调用。
使用websocket进行数据的发送和接受。非常有意思的事情是，如果客户端和服务端都是用go写，用的都是websocket这个对象。函数调用都是一样的，只不过一个写一个读数据而已。

### 3、html5调用

html5页面代码：


```html
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8"/>
    <title>Sample of websocket with golang</title>
    <script src="http://apps.bdimg.com/libs/jquery/2.1.4/jquery.min.js"></script>
    <script>
      $(function() {
        var ws = new WebSocket("ws://localhost:8080/echo");
        ws.onmessage = function(e) {
          $('<li>').text(event.data).appendTo($ul);
        };
        var $ul = $('#msg-list');
        $('#sendBtn').click(function(){
          var data = $('#name').val();
          ws.send(data);
        });
      });
    </script>
</head>
<body>
    <input id="name" type="text"/>
    <input type="button" id="sendBtn" value="send"/>
    <ul id="msg-list"></ul>
</body>
</html>
```

示例展示(参见第三大项:golang官方包实现)

源码地址：https://gitee.com/chenlei888/websocket-demo.git

   


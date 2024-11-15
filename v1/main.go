package board

//package main
//
//import (
//	storetypes "cosmossdk.io/store/types"
//	"encoding/json"
//	"fmt"
//	"github.com/cosmos/cosmos-sdk/codec"
//	sdk "github.com/cosmos/cosmos-sdk/types"
//	"github.com/gorilla/websocket"
//	"net/http"
//	"sync"
//	"time"
//)
//
//// Post 구조체 정의 및 Protobuf 인터페이스 구현
//type Post struct {
//	ID        int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
//	Title     string `protobuf:"bytes,2,opt,name=title,proto3" json:"title"`
//	Content   string `protobuf:"bytes,3,opt,name=content,proto3" json:"content"`
//	Author    string `protobuf:"bytes,4,opt,name=author,proto3" json:"author"`
//	Timestamp int64  `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp"`
//}
//
//func (p *Post) Reset()         { *p = Post{} }
//func (p *Post) String() string { return fmt.Sprintf("Post ID: %d, Title: %s", p.ID, p.Title) }
//func (p *Post) ProtoMessage()  {}
//
//type ForumKeeper struct {
//	storeKey storetypes.StoreKey
//	cdc      codec.Codec
//}
//
//func (k ForumKeeper) setPost(ctx sdk.Context, post *Post) {
//	store := ctx.KVStore(k.storeKey)
//	key := []byte(fmt.Sprintf("post:%d", post.ID))
//	store.Set(key, k.cdc.MustMarshal(post))
//}
//
////	func (k ForumKeeper) CreatePost(ctx sdk.Context, title, content, author string) {
////		post := Post{
////			ID:        1,
////			Title:     title,
////			Content:   content,
////			Author:    author,
////			Timestamp: ctx.BlockTime().Unix(),
////		}
////		k.setPost(ctx, &post)
////	}
////
////	func (k ForumKeeper) CreatePost(ctx sdk.Context, title, content, author string) {
////		post := Post{
////			ID:        time.Now().Unix(),
////			Title:     title,
////			Content:   content,
////			Author:    author,
////			Timestamp: ctx.BlockTime().Unix(),
////		}
////
////		// 상태 저장
////		k.setPost(ctx, &post)
////
////		//
////		//// 상태 변경 시 이벤트 발생
////		//ctx.EventManager().EmitEvent(
////		//	sdk.NewEvent(
////		//		"post_created",
////		//		sdk.NewAttribute("post_id", fmt.Sprintf("%d", post.ID)),
////		//		sdk.NewAttribute("title", post.Title),
////		//		sdk.NewAttribute("author", post.Author),
////		//	),
////		//)
////	}
//func (k *ForumKeeper) CreatePost(ctx sdk.Context, w http.ResponseWriter, r *http.Request) {
//	var newPost Post
//	if err := json.NewDecoder(r.Body).Decode(&newPost); err != nil {
//		http.Error(w, "Invalid input", http.StatusBadRequest)
//		return
//	}
//
//	// 새로운 게시글 생성
//	post := Post{
//		ID:        time.Now().Unix(), // 임의의 ID 할당
//		Title:     newPost.Title,
//		Content:   newPost.Content,
//		Author:    newPost.Author,
//		Timestamp: time.Now().Unix(),
//	}
//
//	// 저장
//	k.setPost(ctx, &post)
//
//	// 성공 응답
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(http.StatusCreated)
//	json.NewEncoder(w).Encode(post)
//}
//
////func (k ForumKeeper) GetPost(ctx sdk.Context, id int64) (*Post, bool) {
////	store := ctx.KVStore(k.storeKey)
////	key := []byte(fmt.Sprintf("post:%d", id))
////	if !store.Has(key) {
////		return nil, false
////	}
////	var post Post
////	k.cdc.MustUnmarshal(store.Get(key), &post)
////	return &post, true
////}
//
//var upgrader = websocket.Upgrader{
//	CheckOrigin: func(r *http.Request) bool { return true },
//}
//
//var clients = make(map[*websocket.Conn]bool)
//var broadcast = make(chan string)
//var mu sync.Mutex
//
//// WebSocket 연결을 처리하는 함수
//func handleWebSocket(w http.ResponseWriter, r *http.Request) {
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		fmt.Println("Error upgrading to WebSocket:", err)
//		return
//	}
//	defer conn.Close()
//
//	// 클라이언트를 등록
//	mu.Lock()
//	clients[conn] = true
//	mu.Unlock()
//
//	// 연결이 닫힐 경우 클라이언트를 제거
//	defer func() {
//		mu.Lock()
//		delete(clients, conn)
//		mu.Unlock()
//	}()
//
//	for {
//		// 메시지를 받아서 처리할 수 있지만 현재는 필요하지 않음
//		_, _, err := conn.ReadMessage()
//		if err != nil {
//			fmt.Println("Error reading message:", err)
//			break
//		}
//	}
//}
//
//// 게시글이 생성되었을 때 WebSocket 클라이언트에게 메시지 전송
//func broadcastMessage() {
//	for {
//		message := <-broadcast
//		mu.Lock()
//		for client := range clients {
//			if err := client.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
//				fmt.Println("Error sending message:", err)
//				client.Close()
//				delete(clients, client)
//			}
//		}
//		mu.Unlock()
//	}
//}
//
//func main() {
//	http.HandleFunc("/ws", handleWebSocket)
//	go broadcastMessage()
//
//	// 임의의 이벤트 발생 (게시글 생성 시 메시지 전송)
//	go func() {
//		for {
//			time.Sleep(5 * time.Second)
//			broadcast <- `{"event": "post_created", "post_id": 1, "title": "Hello Cosmos", "author": "user1"}`
//		}
//	}()
//
//	fmt.Println("WebSocket server started on :8080")
//	http.ListenAndServe(":8080", nil)
//	// Cosmos SDK 설정
//	//storeKey := storetypes.NewKVStoreKey("forum")
//	//interfaceRegistry := types.NewInterfaceRegistry()
//	//cdc := codec.NewProtoCodec(interfaceRegistry)
//	//
//	//// ForumKeeper 인스턴스 생성
//	//forumKeeper := ForumKeeper{
//	//	storeKey: storeKey,
//	//	cdc:      cdc,
//	//}
//	//metricGatherer := metrics.NewNoOpMetrics() // 메트릭 수집기 생성
//	//logger := log.NewNopLogger()
//	//// MemDB 데이터베이스 초기화
//	//db := db.NewMemDB()
//	//cms := store.NewCommitMultiStore(db, logger, metricGatherer)
//	//cms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
//	//if err := cms.LoadLatestVersion(); err != nil {
//	//	panic(fmt.Sprintf("failed to load latest version: %v", err))
//	//}
//	//
//	//// 블록 헤더 생성 및 컨텍스트 생성
//	//header := tmproto.Header{
//	//	Time: time.Now(),
//	//}
//	//
//	//ctx := sdk.NewContext(cms, header, false, logger)
//	//
//	//// 게시글 생성 및 조회 테스트
//	//forumKeeper.CreatePost(ctx, "Hello Cosmos", "This is a test post", "user1")
//	//post, found := forumKeeper.GetPost(ctx, 1)
//	//if found {
//	//	fmt.Printf("Post found: %+v\n", post)
//	//} else {
//	//	fmt.Println("Post not found")
//	//}
//}

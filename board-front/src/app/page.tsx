"use client"
import React, { useState, useEffect } from 'react';

function Home() {
  const [socket, setSocket] = useState(null);
  const [postData, setPostData] = useState(null);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [author, setAuthor] = useState("");
  const [postId, setPostId] = useState(""); // 조회할 게시글 ID

  // 웹소켓 연결
  useEffect(() => {
    const ws = new WebSocket("ws://localhost:8080/ws");
    setSocket(ws);

    ws.onopen = () => {
      console.log("WebSocket connected");
    };

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      console.log("Received message:", data);
      setPostData(data);
    };

    ws.onerror = (error) => {
      console.error("WebSocket Error:", error);
    };

    ws.onclose = () => {
      console.log("WebSocket disconnected");
    };

    // 연결 종료 시 cleanup
    return () => {
      ws.close();
    };
  }, []);

  // 게시글 생성
  const createPost = () => {
    if (socket && title && content && author) {
      const post = { action: "insert",title, content, author };
      const postMessage = JSON.stringify(post);
      socket.send(postMessage);
      console.log("Sent post:", post);
    }
  };
  // 게시글 조회
  const getPost = () => {
    if (socket && postId) {
      const request = { action: "retrieve", id: postId };
      socket.send(JSON.stringify(request));
      console.log("Sent get_post request:", request);
    }
  };


  return (
      <div className="App">
        <h1>Post Creation and WebSocket Example</h1>

        {/* 게시글 생성 폼 */}
        <div>
          <input
              type="text"
              placeholder="Title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
          />
          <textarea
              placeholder="Content"
              value={content}
              onChange={(e) => setContent(e.target.value)}
          />
          <input
              type="text"
              placeholder="Author"
              value={author}
              onChange={(e) => setAuthor(e.target.value)}
          />
          <button onClick={createPost}>Create Post</button>
        </div>

        {/* 서버로부터 받은 게시글 데이터 표시 */}
        {postData && (
            <div>
              <h2>Post Created:</h2>
              <p>Title: {postData.title}</p>
              <p>Content: {postData.content}</p>
              <p>Author: {postData.author}</p>
            </div>
        )}
        {/* 게시글 조회 폼 */}
        <div>
          <input
              type="text"
              placeholder="Post ID"
              value={postId}
              onChange={(e) => setPostId(e.target.value)}
          />
          <button onClick={getPost}>Get Post</button>
        </div>
      </div>
  );
}

export default Home;

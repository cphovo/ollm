# ollm

A simple demo providing series of LLM's API.

## Todo

- [ ] Reset cookies while `Error: bing explicit error: value: Throttled; message: Request is throttled.`
- [ ] Auto refresh bing cookies
- [x] Rebuild API
- [ ] Simple input/output

## Usage

Start the service and send the request: 

```shell
curl 'http://127.0.0.1:8080/v1/chat/completions' \
--header 'Content-Type: application/json' \
--data '{
    "model": "Creative",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful assistant."
      },
      {
        "role": "user",
        "content": "讲一个有趣的笑话"
      }
    ]
  }'
```

then you will receive the data in the following format:

```json
{
    "id": "chatcmpl-123",
    "object": "chat.completion",
    "created": 1711986244,
    "model": "Creative",
    "system_fingerprint": "fp_44709d6fcb",
    "choices": [
        {
            "index": 0,
            "message": {
                "content": "当然可以！这里有一个：\n\n一个程序员的妻子对他说：“去商店买一条面包，如果有鸡蛋，买六个。”\n他回来了，带着六条面包。\n他的妻子问：“为什么买了这么多面包？”\n他回答：“因为他们有鸡蛋。” 😄\n\n希望这个笑话能让你笑一笑！如果你想听更多笑话或者需要其他帮助，请告诉我！\n",
                "role": "assistant"
            },
            "finish_reason": "stop"
        }
    ],
    "usage": {
        "prompt_tokens": 1024,
        "completion_tokens": 1024,
        "total_tokens": 2048
    }
}
```

Supported Models:
- `Creative`
- `Balanced`
- `Precise`
- `gpt-3.5-turbo`
- `kimi`
- `gemini-pro`

## Thanks

This demo reference juzeon's open source project ([SydneyQt](https://github.com/juzeon/SydneyQt)), many thanks!🙏
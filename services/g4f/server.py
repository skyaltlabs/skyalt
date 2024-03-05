from http.server import BaseHTTPRequestHandler, HTTPServer
import sys, traceback
import json
from g4f.client import Client



class MyHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        received_json = json.loads(post_data.decode('utf-8'))
        #print(f"Received JSON: {received_json}")

        model = received_json['model']
        messages = json.loads(received_json['messages'])
        if model == "":
            model = "gpt-4-turbo"

        answer = ""
        if model != "" and len(messages) > 0:
            client = Client()
            stream = client.chat.completions.create(
                model=model,
                messages=messages,
                stream=True,
            )

            for chunk in stream:
                if chunk.choices[0].delta.content:
                    answer += chunk.choices[0].delta.content
                    print(chunk.choices[0].delta.content, flush=True, end='')

        js = json.dumps(answer)
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(bytes(js, "utf8"))

if __name__ == "__main__":
    port = 8080
    if len(sys.argv) > 1:
        port = int(sys.argv[1])
    
    print('args: port =', port)
    server_address = ('', port)
    httpd = HTTPServer(server_address, MyHandler)
    print('HTTP server is runnning on port', port)
    httpd.serve_forever()


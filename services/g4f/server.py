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

        role = received_json['role']
        model = received_json['model']
        prompt = received_json['prompt']
        if role == "":
            role = "user"
        if model == "":
            model = "gpt-4-turbo"

        answer = ""
        if prompt != "":
            client = Client()
            stream = client.chat.completions.create(
                model=model,
                messages=[{"role": role, "content": prompt}],
                stream=True,
            )

            for chunk in stream:
                if chunk.choices[0].delta.content:
                    answer += chunk.choices[0].delta.content
                    print(chunk.choices[0].delta.content, flush=True, end='')

        res = {}
        res['role'] = role
        res['model'] = model
        res['prompt'] = prompt
        res['answer'] = answer
        js = json.dumps(res)

        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(bytes(js, "utf8"))

if __name__ == "__main__":
    port = 8080
    if len(sys.argv) > 1:
        port = int(sys.argv[1])
    
    server_address = ('', port)
    httpd = HTTPServer(server_address, MyHandler)
    print('HTTP server is runnning on port', port)
    httpd.serve_forever()


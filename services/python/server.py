from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class MyHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        received_json = json.loads(post_data.decode('utf-8'))
        #print(f"Received JSON: {received_json}")

        code = received_json['code']
        attrs = received_json['attrs']

        exec(code, {}, attrs)

        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(bytes(json.dumps(attrs), "utf8"))

def run(server_class=HTTPServer, handler_class=MyHandler):
    server_address = ('', 8092)
    httpd = server_class(server_address, handler_class)
    print('Starting httpd...')
    httpd.serve_forever()

if __name__ == "__main__":
    run()


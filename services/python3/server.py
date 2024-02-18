from http.server import BaseHTTPRequestHandler, HTTPServer
import sys, traceback
import json


class MyHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        received_json = json.loads(post_data.decode('utf-8'))
        #print(f"Received JSON: {received_json}")

        code = received_json['code']
        attrs = received_json['attrs']
        errStr = ""
          
        try:
            exec(code, {'json': json}, attrs)
        except SyntaxError as err:
            error_class = err.__class__.__name__
            detail = err.args[0]
            line_number = err.lineno
            errStr = "%s at line %d: %s" % (error_class, line_number, detail)
            print(err)
        except Exception as err:
            error_class = err.__class__.__name__
            detail = err.args[0]
            cl, exc, tb = sys.exc_info()
            line_number = traceback.extract_tb(tb)[-1][1]
            errStr = "%s at line %d: %s" % (error_class, line_number, detail)
            print(err)

        res = {}
        res['attrs'] = attrs
        res['err'] = errStr
        js = json.dumps(res)

        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        self.wfile.write(bytes(js, "utf8"))

def run(server_class=HTTPServer, handler_class=MyHandler):
    port = 8080
    if len(sys.argv) > 1:
        port = int(sys.argv[1])
    
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    print('HTTP server is runnning on port', port)
    httpd.serve_forever()

if __name__ == "__main__":
    run()


from http.server import BaseHTTPRequestHandler, HTTPServer
import sys, traceback
import json


class Text: #render text on screen
	def __init__(self):
		self.class_name = "Text"
		self.grid = [0, 0, 1, 1]	#coordinates on screen
		self.label = "example text"

class Editbox: #render editbox on screen
	def __init__(self):
		self.class_name = "Editbox"
		self.grid = [0, 0, 1, 1]	#coordinates on screen
		self.enable = True
		self.value = "example text"
		self.empty = 0
		self.finished = 0

class Button:
	def __init__(self):
		self.class_name = "Button"
		self.grid = [0, 0, 1, 1]	#coordinates on screen
		self.enable = True
		self.label = "Click me!"
		self.clicked = False


class MyHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        
        received_json = json.loads(post_data.decode('utf-8'))
        #print(f"Received JSON: {received_json}")

        code = received_json['code']
        errStr = ""

        resValues = {} 
        if code != "":          
            try:
                attrs = {}
                exec(code, {'Text': Text, 'Button': Button}, attrs)
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


            for key, v in attrs["sa"].__dict__.items():
                if hasattr(v, "__dict__"):
                    resValues[key] = v.__dict__
                else:
                    resValues[key] = v

        res = {}
        res['attrs'] = resValues
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


## SkyAlt
Build node-based, local-first, and LLM-powered apps.


<p align="center">
<img src="https://github.com/skyaltlabs/skyalt/blob/main/screenshots/screenshot_2024-3-13_21-33-40_small.png?raw=true" style="border:1px solid LightGrey" />
</p>



## Videos
- Demo 1
    - Introduction: https://www.youtube.com/watch?v=655nz44RgLY
    - Apps from scratch(no audio): https://www.youtube.com/watch?v=YDIxueaJTeo



## Node-based
In Unix, everything is a file. In SkyAlt, everything is a node.
- Node can be GUIs(button, slider, map, calendar, etc.) or programs(LLM, SQLite, Python, etc.).
- Nodes are composed into graphs. 
- Graph can be put into other graphs. 



## LLM-powered
SkyAlt is not about AI research, but **interface research and development**. LLM is the new programming paradigm that can make human-computer interactions way more efficient, natural and frictionless.

SkyAlt goal is to offer a holistic and frictionless experience. Building an app means 2 things:
- Put GUIs on layout
- Write a list of features in natural language


## Local-first
Most of today's apps run in a browser as Software as a Service. Here's the list of problems you may experience:
- delay between client and server
- none or simple export
- hard to migrate between clouds
- data disappear(music playlist, etc.)
- data was tampered
- new SaaS version was released and you wanna keep using the older one
- no offline mode
- SaaS shut down
- price goes up
- 3rd party can access your data

SkyAlt solves them with Local-first computing. The biggest advantages can be summarized as:
- quick responses
- works offline
- ownership
- privacy
- works 'forever' + run any version

For endurance, SkyAlt uses only well-known and open formats SQLite and JSON.



## Build
SkyAlt is written in Go language. You can install golang from here: https://go.dev/doc/install

Dependencies:
<pre><code>go get github.com/mattn/go-sqlite3
go get github.com/veandco/go-sdl2/sdl
go get github.com/veandco/go-sdl2/gfx
go get github.com/go-gl/gl/v2.1/gl
go get github.com/fogleman/gg
go get github.com/go-audio/audio
go get github.com/golang/freetype/truetype
</code></pre>

SkyAlt:
<pre><code>git clone https://github.com/skyaltlabs/skyalt
cd skyalt
go build
./skyalt
</code></pre>

Service LLama.cpp(~100MB):
<pre><code>cd services
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp
make
</code></pre>

Service Whisper.cpp(~30MB):
<pre><code>cd services
git clone https://github.com/ggerganov/whisper.cpp
cd whisper.cpp
make
</code></pre>



## Inspiration
- LLM OS:
    - https://twitter.com/karpathy/status/1723140519554105733
    - https://www.youtube.com/watch?v=zjkBMFhNj_g&t=2540s

- Software 2.0:  https://www.youtube.com/watch?v=y57wwucbXR8
    - Andrej Karpathy: https://www.youtube.com/watch?v=UJc8UpClSUQ
    - Chris Lattner: https://www.youtube.com/watch?v=orY5aLMDU-I



## Acknowledgements
- [Go](https://go.dev/): The Go programming language
- [SDL](https://www.libsdl.org/) + [Go binding](https://github.com/veandco/go-sdl2): Cross-platform graphics hardware library
- [SQLite](https://www.sqlite.org/) + [Go binding](https://github.com/mattn/go-sqlite3): Database engine

- [Llama.cpp](https://github.com/ggerganov/llama.cpp): LLM inference in C/C++
- [Whisper.cpp](https://github.com/ggerganov/whisper.cpp): Port of OpenAI's Whisper model in C/C++

- [Inter](https://github.com/rsms/inter): The Inter font family



## Author
Milan Suk

Email: milan@skyalt.com

Twitter: https://twitter.com/milansuk/

**Sponsor**: https://github.com/sponsors/MilanSuk

*Feel free to follow or contact me with any idea, question or problem.*



## Contributing
Your feedback and code are welcome!

For bug report or question, please use [GitHub's Issues](https://github.com/skyaltlabs/skyalt/issues)

SkyAlt is licensed under **Apache v2.0** license. This repository includes 100% of the code.

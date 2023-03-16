# wordsplit

Build:

    $ go build -o ./cmd/wordsplit .

Or, install directly:

    $ go install github.com/fsufitch/wordsplit/cmd/wordsplit

Usage:

    $ ./wordsplit -h
    Usage of ./wordsplit:
      -f string
            words file to use; default from WORDS_FILE (default "")
      -nw int
            maximum valid nonword length (default 3)
      -s    use stdin to get words to split; otherwise, split positional args
      -w int
            minimum valid word length (default 3)

Use with `-f` to pass in a words file ([get words_alpha.txt from here](https://github.com/dwyl/english-words)).    

    $ ./wordsplit -f words_alpha.txt helloworld theholygrail
    helloworld -> ["hell","(owo)","rld"]
    helloworld -> ["hello","world"]
    theholygrail -> ["the","hol","(yg)","rail"]
    theholygrail -> ["the","hol","(ygr)","ail"]
    theholygrail -> ["the","holy","grail"]
    
Respects `WORDS_FILE` variable if none is passed in.

    $ export WORDS_FILE="$(pwd)/words_alpha.txt"

    $ ./wordsplit coolwordsplitting
    coolwordsplitting -> ["coo","(lw)","ord","splitting"]
    coolwordsplitting -> ["coo","(lw)","ord","spl","(itt)","ing"]
    coolwordsplitting -> ["coo","(lw)","ord","split","ting"]
    coolwordsplitting -> ["coo","lwo","(rds)","pli","(tt)","ing"]
    coolwordsplitting -> ["cool","(wor)","dsp","lit","ting"]
    coolwordsplitting -> ["cool","word","splitting"]
    coolwordsplitting -> ["cool","word","spl","(itt)","ing"]
    coolwordsplitting -> ["cool","word","split","ting"]
    coolwordsplitting -> ["cool","words","pli","(tt)","ing"]

Use with `-nw` or `-w` to narrow down what you're looking for.

    $ ./wordsplit -nw 0 helloworld theholygrail
    helloworld -> ["hello","world"]
    theholygrail -> ["the","holy","grail"]

Take input from stdin by not passing any positional args.

    $ printf 'examplestandard\ninputwith\nmultiplelines sometimeswithspaces' | ./wordsplit -nw 0
    examplestandard -> ["example","standard"]
    examplestandard -> ["example","stan","dard"]
    examplestandard -> ["examples","tan","dard"]
    inputwith -> ["input","with"]
    multiplelines -> ["multiple","lines"]
    sometimeswithspaces -> ["some","time","swith","spaces"]
    sometimeswithspaces -> ["some","times","with","spaces"]
    sometimeswithspaces -> ["sometime","swith","spaces"]
    sometimeswithspaces -> ["sometimes","with","spaces"]

No Go, but have Docker?

    $ docker build -qt wordsplit .

    $ docker run wordsplit -nw 0 correcthorsebatterystaple
    correcthorsebatterystaple -> ["cor","rect","horse","battery","staple"]
    correcthorsebatterystaple -> ["correct","horse","battery","staple"]

    $ printf 'workswithstreamstoo' | docker run -i wordsplit -nw 0
    workswithstreamstoo -> ["work","swith","streams","too"]
    workswithstreamstoo -> ["works","with","streams","too"]

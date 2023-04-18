# NES Emulation Study

This repo is some experiments into emulation world. I'm learning how to build game consoles emulators, starting with NES. Feel free to check it out and comment.

### How to execute
I'm building using Go, so you need to compile it first
```
go build nes-emu
```
Then just execute indicating a NES ROM file path
```
./nes-emu --rom nestest.nes
```

### Todo
- [x] Implement all CPU instructions
- [x] Implement PPU foreground and background rendering
- [x] Implement Mapper 000
- [x] Implement one pulse audio channel
- [ ] Implement more mappers
- [ ] Implement more audio channels

### References

This study is heavily inspired in [Javidx9](https://github.com/OneLoneCoder) YouTube's NES series.

Also NESDev Wiki is a very useful resource https://www.nesdev.org/wiki/NES_reference_guide
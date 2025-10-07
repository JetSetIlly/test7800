Test7800 is an experimental emulator for the Atari 7800. It's not complete and is missing some important features but it plays many of the 7800 ROM files that are available. 

It supports a78 files, including non-bankswitching regular "flat" ROM files and several different bankswitching "supergame" ROM files. While it does not emulate all conglomerate cartridge hardware configurations, the POKEY chip and many of its layouts are supported.

The 6502, TIA and RIOT emulation is taken from [Gopher2600](https://github.com/JetSetIlly/Gopher2600) and is therefore well tested. The implemenation of the MARIA is new to this project.

### Basic Usage

Running the program from the desktop icon will open a file selection dialog. Opening a 7800 ROM will cause the emulation window to open.

Only one-button and two-button joysticks for the first player are supported for now. Also, the keyboard must be used. The cursor keys control the stick and the space bar is the fire button. The 'B' key acts as the second fire button.

A command line debugger is available if the program is run from a terminal. In this case, the ROM file should be specified as part of the command line (eg. `test7800 centipede.a78`). The debugger will start in a halted state. To run the emulation from this point, type `RUN` in the terminal.

Pressing `Ctrl-C` when the emulation is running will cause it to halt and to resume the debugger.

The debugger is currently very basic and missing a lot of features. However, some useful commands include `STEP`, `RESET`, `CPU`, `MARIA`, `DL`, `DLL`, `VIDEO`, `INPTCTRL`, `RAM7800`, `RAMRIOT`. 

A useful option to the program is the `-overlay` argument. (eg. `test7800 -overlay centipede.a78)`). This adds an additional overlay to the TV screen, showing the state of the MARIA at each point in the display. The colours in the overlay are as follows

| Colour | Meaning |
|--------|---------|
| Red  | DMA Active |
| Blue | WSYNC Active |
| Green | CPU in Interrupt |

By default, the NTSC BIOS is used. To select a PAL BIOS use the `-tv` argument (or `-spec` argument):

```test7800 -tv=pal centipede.a78```

If you want the emulation to ignore the BIOS startup routine use the `-bios` option:

```test7800 -bios=false centipede.a78```

### Limitations and Future

This emulation was developed in order to gain an understanding of the Atari 7800 and so is missing many features. The debugger in particular only exists so that I could more easily debug the emulator itself during development. It probably isn't that useful for ROM development as it currently exists.

The ultimate plan is to combine this 7800 emulation with [Gopher2600](https://github.com/JetSetIlly/Gopher2600). However, it is proving to be a convenient stand-alone emulator and so is being released for general consumption. The integration with Gopher2600 will happen but probably not any time soon.

#### Performace

Because the emulator is currently only a test for future ideas it has not been written with performance in mind. No optimisation or perfomance analysis has been performed, with the exception of the automated use of profile guided optimisation. None-the-less, the emulator should run well on reasonably modern hardware. For comparison purposes, the development machine has an `i3-3225` CPU running at 3.30GHz.

### Resources Used

References to "7800 Software Guide" in comments are referring to [this wiki page](https://7800.8bitdev.org/index.php/7800_Software_Guide). This wiki is part of a larger set of 7800 related articles: [Atari 7800 Development Wiki Home](https://7800.8bitdev.org/index.php/Main_Page)

[7800 PAL OS source code](https://forums.atariage.com/index.php?app=core&module=system&controller=redirect&url=https://web.archive.org/web/20200831200403/http://www.atarimuseum.com/videogames/consoles/7800/games/&key=e73e4f017a3c7a18a6715c7cd61fadc2936d952c7f60ee7d37484620d0b540bb&email=1&type=notification_new_comment)

[7800 Hardware Facts](https://forums.atariage.com/topic/224025-7800-hardware-facts)

[Atari 7800 Difficulty Switches Guide](https://forums.atariage.com/topic/235913-atari-7800-difficulty-switches-guide/)

[Common Emulator Development Issues](https://7800.8bitdev.org/index.php/Common_Emulator_Development_Issues)

[Has Anyone Worked on an FPGA Atari 7800?](https://forums.atariage.com/topic/214384-has-anyone-worked-on-an-fpga-atari-7800/page/2/#comment-2807000)
	
[A78 Primer](https://forums.atariage.com/topic/333208-old-world-a78-format-10-31-primer/)

[A78 Header Specification](https://7800.8bitdev.org/index.php/A78_Header_Specification)

[Bank Switching Specifics](https://7800.8bitdev.org/index.php/ATARI_7800_BANKSWITCHING_GUIDE)

[Two Button Controllers](https://forums.atariage.com/topic/127162-question-about-joysticks-and-how-they-are-read/#findComment-1537159)

[POKEY C012294 Documentation](https://7800.8bitdev.org/index.php/POKEY_C012294_Documentation)

[Altirra Hardware Reference Manual, Chapter 5](https://www.virtualdub.org/downloads/Altirra%20Hardware%20Reference%20Manual.pdf)

[POKEY implementation in the Altirra emulator](https://www.virtualdub.org/altirra.html)

[Original Atari document for POKEY](http://visual6502.org/images/C012294_Pokey/pokey.pdf)

[POKEY schematics](https://atarimuseum.ctrl-alt-rees.com/whatsnew/2016-NOV-2.html)

### Acknowledgements

Zachary Scolaro helped with the MARIA emulation, particular to get it off the ground when I wasn't sure about it at all. And Rob Tuccitto has provided much advice and links to information about the 7800 internals. Thanks to both.

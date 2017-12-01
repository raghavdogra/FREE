PROGNAME = inference

CC = g++
EXT = .cc

LIBDIRS =
INCDIRS = ../tbb/include
CFLAGS = -std=c++1y -O3 -g3 -Wall
CPPFLAGS =
LDFLAGS = -Wl,-rpath,../tbb -L../tbb
LIBS = tbb

##########################################################

CXX = $(CC)
CXXFLAGS = $(CFLAGS)

LDFLAGS += $(addprefix -L,$(LIBDIRS))
CPPFLAGS += $(addprefix -I,$(INCDIRS))

.PHONY: all strip clean depends

sources := $(subst ./,,$(shell find . -name \*$(EXT)))
deps := $(addprefix .deps/,$(sources:$(EXT)=.d))
objects := $(sources:$(EXT)=.o)

all: $(deps) $(PROGNAME)

$(PROGNAME): $(objects)
	$(CC) $(LDFLAGS) -o $@ $(filter-out %.d,$^) $(addprefix -l,$(LIBS))

../tbb/libtbb.so: ../tbb
	make -C ../tbb -j
	find ../tbb -name libtbb.so\* -exec ln -s {} ../tbb \;

../tbb:
	cd .. && git clone https://github.com/01org/tbb.git

%.o: %$(EXT) ../tbb
	$(CXX) -o $@ -c $(CPPFLAGS) $(CXXFLAGS) $<

clean:
	rm -f $(PROGNAME) $(PROGNAME).core core
	rm -f $(objects)
	csh -c "rm -f $(patsubst %.tex,%,$(docsources)).{dvi,ps,pdf,aux,log,bbl,blg}"
	rm -f $(patsubst %.fig,%.eps,$(figs))

redep:
	@rm -fr .deps

veryclean: clean redep

file = $(patsubst .deps/%.d,%,$@)
$(deps):
	@echo "Generating dependencies ($@)"
	@mkdir -p $(shell dirname $@)
	@sh -ec "$(CC) -MM $(CPPFLAGS) $(file)$(EXT) | sed '1s|^.*:|$@ $(file).o:|g' > $@"

-include $(deps) /dev/null

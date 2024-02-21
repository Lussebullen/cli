# Report for assignment 3

## Project

**Name**: docker-CLI

**URL**: https://github.com/docker/cli

docker is a popular container tool used by many, it's a light weight virtulization-like tool. The project is an open source project of the docker CLI tool, used by docker.


# Onboarding experience 

Overall the Docker CLI codebase is quite mature, resulting in a smooth onboarding experience, although this is largely dependent on your OS as reflected in the descriptions below. 
The codebase contains a generic README.md with information on the overarching project, build instructions etc. as well as a [TESTING.md](https://github.com/docker/cli/blob/master/TESTING.md) file outlining testing conventions, and a [CONTRIBUTING.md](https://github.com/docker/cli/blob/master/CONTRIBUTING.md) file outlining standards and conventions for contributors.
Furthermore, the project makes use of code coverage tools, allowing us to more easily navigate the code-base to locate areas in need of improvement, this can be done using their [Coverage Report](https://app.codecov.io/gh/docker/cli).

## Windows

The README in the docker CLI repo only contains a few terminal commands, none of which works natively on Windows. When asking ChatGPT how to only get docker CLI working on Windows, it refers to the docs.docker.com site, where it is stated that you need docker desktop to run the docker CLI on Windows. 

After installing docker desktop, the `docker` command is automatically added to PATH and the command can be run through PowerShell/cmd. Additionally, 'secure boot' has to be active in BIOS for docker desktop to work and the command `service docker start` has to be used since WSL does not use systemctl.

However, the repo uses a make file and the `make` command is not native to Windows and can not be run. A popular make solution is to download GNU make, a Windows version that enables make in Windows-shell. However, the make file still contained commands that were not supported by GNU make. The last option was to manually reconfigure the whole make file so that it runs on Windows. At this point, Windows was abandoned and instead, WSL was used. In WSL everything worked flawlessly for all aspects of the project.

## Linux

### (a) Did you have to install a lot of additional tools to build the software? 


To build the project on linux systems there are essentially three requirements:
docker
docker-buildx
make
After installing and setting up these programs appropriately, the build script worked without issue. 

### (b) Were those tools well documented?

Docker is well-documented, and has a large and active community, so there are numerous guides covering any basic issue you might encounter. This extends to the build-tool.
Similarly, make is a widely used build-tool with extensive documentation.

### (c) Were other components installed automatically by the build script?

Yes, in a manner of speaking. Docker CLI is developed using Docker, so when building the project we are launching a Docker container with all remaining prerequisites installed, so you could say that almost everything is installed by the build script, only not into our local system, but rather we run a container with all the prerequisites.

### (d) Did the build conclude automatically without errors?
Yes, no errors are encountered in the build process, which is to be expected when it is run from a container provided by the developers.

### (e) How well do examples and tests run on your system(s)?
If you are in the project folder and run ```make help``` you get an overview of available commands, running the tests from here works smoothly and generates a coverage report. 
Most crucially there is a make command that runs an interactive session with a development container from where you can run all the other make commands without issue.
This essentially renders it system - agnostic, so it works great.


# Complexity

## Question 1

**What are your results? Did everyone get the same result? Is there something that is unclear? If you have a tool, is its result the same as yours?**

For our complexity analysis we ran Lizard on our code base, and chose our complex functions based on the cyclomatic complexity number (CCN) reported back by Lizard. Below you can see an overview of our functions of choice.

```
| Function         | CCN | Owner  | Location                                    |
|------------------|-----|--------|---------------------------------------------|
| extractVariable  | 10  | Mathias| cli/cli/compose/template/template.go       	|
| importTar        | 14  | Adam   | cli/cli/context/store/store.go             	|
| interactiveExec  | 10  | Emil   | cli/cli/command/container/exec.go          	|
| prettyPrintInfo  | 9   | Omar   | cli/cli/command/system/info.go             	|
| newPlugin        | 14  | Emir   | cli/cli-plugins/manager/plugin.go           |
```

### Results

In this section we go through the aggregated results. It should be noted that we strive to use the same computation method as Lizard, meaning the “Cyclomatic Number”, so we create a strongly connected control flow graph, i.e. with the exit nodes connecting back to the entry node. 
We use the formula CCN = E - N - P where E is the number of edges, N the number of nodes and P the number of connected components.

```
| Function        | Lizard | Mathias | Emil |
|-----------------|--------|---------|------|
| extractVariable | 10 	   | 10      | 10   |
| importTar       | 14 	   | 13      | 13   |
| interactiveExec | 10 	   | 10      | 9    |
| prettyPrintInfo | 9  	   | 9       | 8    |
| newPlugin       | 14 	   | 14      | 14   |
```

Note here, there are three points of interest. 
How our calculation differs from Lizard in importTar.
We suspect this different comes from the Golang convention of using “for {}” as infinite loops without any condition to check, we assume that Lizard has counted this for-loop as a regular for loop that checks a condition to break at some point, and thus over-counted by one.
How Mathias and Emils calculations differ in interactiveExec and prettyPrintInfo.
This difference lies in Mathias unfolding && and || statements into separate nodes, and thus getting +1 complexity for each such occurrence.

## Question 2

**Are the functions/methods with high CC also very long in terms of LOC?**

In the below table you see an overview of the correspondence between cyclomatic complexity and associated lines of code. As you can see, the correspondence is quite clear, which is not too surprising, since more lines of code can introduce more branches in the control flow and thus higher complexity.

```
| Function         | CCN | LOC |
|------------------|-----|-----|
| extractVariable  | 10  | 35  |
| importTar        | 14  | 48  |
| interactiveExec  | 10  | 42  |
| prettyPrintInfo  | 9   | 27  |
| newPlugin        | 14  | 56  |
```

## Question 3

**What is the purpose of these functions? Is it related to the high CC?**

- Function 1: extractVariable

This function essentially does string-parsing using regular expressions; the parsing behavior differs depending on the presence of pre-defined substrings. This naturally leads to the use of a switch statement, and generally this kind of parsing can be quite complex.

- Function 2: importTar

Imports data from a TAR archive, and performs various validation checks to ensure security / integrity. Strict security requirements can lead to extensive validation checking which in turn is responsible for high complexity.



- Function 3: interactiveExec

Sets up and starts a console window, where commands can be executed interactively. It is called as a part of another functionality, when “exec” commands are to be run, but the environment is not specified. In that case, it is run as an interactive environment as a default case which this function handles.

- Function 4: prettyPrintInfo

Prints information about a client and server in a neat way. Additionally, eventual errors during printing are handled and communicated if they occur.

- Function 5: newPlugin

Checks if a given candidate for a new plugin is valid. If a candidate fails a test, “Plugin.Err” is set but a plugin is still returned. Only if there are errors is a plugin not returned.

## Question 4

**If your programming language uses exceptions: Are they taken into account by the tool? If you think of an exception as another possible branch (to the catch block or the end of the function), how is the CC affected?**

Golang doesn’t have traditional error handling, it returns the error status to the user along with the result, and you manually handle the error if not nil. 
Which means all the handling is explicit in the code and thus handled by the Lizard tool.

## Question 5

**Is the documentation of the function clear about the different possible outcomes induced by different branches taken?**

The code-base in general does not make frequent use of comments, most description is inferred from the function names and general code structure - which is pretty clear, but would be much more friendly to newcomers of the code-base had they included comments. 
Thus, the documentation is not clear about the different possible outcomes related to taking different branches.


# Coverage
## What is the quality of your own coverage measurement? Does it take into account ternary operators (condition ? yes : no) and exceptions, if available in your language?
The coverage tool we implemented is limited to a large degree since we followed the simple outline provided in the assignment. Our tool works by having us manually count the number of branches inside a function, give each branch an ID and then allocated a hashmap of `int:bool` pairs, where the ID is the key and where each ID's value is set to `false` in the beginning. We then execute all available tests for the function we are testing that flip an ID to `true` if the branch is reached in any of the tests. After executing all tests we pass the hashmap to a function we have written that writes all `key-value` pairs to file as well as the total branch coverage percentage. Since we worked in `golang` we did not have to take the ternary operator and exceptions in mind since the language does not include them. 

## What are the limitations of your tool? How would the instrumentation change if you modify the program?
Our tool is limited as stated above and more specifically for two reasons:
- It is not automated. We must specifically find all brances inside a function and add a line in the branch that enables a flag inside the hashmap. The hashmap must also be included in the code, as well as the function we wrote that takes a hashmap and writes results to the drive. 
- We must manually add code to the existing code base, something that a sophisticated tool would not do for example. 
If one modify a given function the instrumentation is affected because any additions/removals in the program would mean that we have to adjust the instrumentation inside the file before we could test the coverage again. 

## If you have an automated tool, are your results consistent with the ones produced by existing tool(s)?
Our results are not consistent because our tool calculates branch coverage while the automated tool we have used calculates statement coverage.



## Coverage before and after
When running the repos test-coverage test with `make test-coverage` the coverage is increased as listen down below.

Before :
- cli/opts: 66.9% (Emir)
- cli/compose/convert 79.6% (Emil)
- cli/compose/convert 80.6% (Mathias)
- cli/compose/convert 81.5% (Adam&Mathias)
- cli/compose/convert 82.2% (Adam)
- cli/compose/convert 83.6% (Omar)
- cli/command 47.8% (Omar)

After : 
- cli/opts: 68.4% (Emir)
- cli/compose/convert 80.6% (Emil)
- cli/compose/convert 81.5% (Mathias)
- cli/compose/convert 82.2% (Adam&Mathias)
- cli/compose/convert 83.6% (Adam)
- cli/compose/convert 84.3% (Omar)
- cli/command 48.4% (Omar)


# Task 3: Refactoring plan

Is the high complexity you identified really necessary? Is it possible to split up the code (in the five complex
functions you have identified) into smaller units to reduce complexity? If so, how would you go about this?

## extractVariable (Mathias)

```go
func extractVariable(value any, pattern *regexp.Regexp) ([]extractedValue, bool) {
    sValue, ok := value.(string)
    if !ok {
   	 return []extractedValue{}, false
    }
    matches := pattern.FindAllStringSubmatch(sValue, -1)
    if len(matches) == 0 {
   	 return []extractedValue{}, false
    }
    values := []extractedValue{}
    for _, match := range matches {
   	 groups := matchGroups(match, pattern)
   	 if escaped := groups["escaped"]; escaped != "" {
   		 continue
   	 }
   	 val := groups["named"]
   	 if val == "" {
   		 val = groups["braced"]
   	 }
   	 name := val
   	 var defaultValue string
   	 switch {
   	 case strings.Contains(val, ":?"):
   		 name, _ = partition(val, ":?")
   	 case strings.Contains(val, "?"):
   		 name, _ = partition(val, "?")
   	 case strings.Contains(val, ":-"):
   		 name, defaultValue = partition(val, ":-")
   	 case strings.Contains(val, "-"):
   		 name, defaultValue = partition(val, "-")
   	 }
   	 values = append(values, extractedValue{name: name, value: defaultValue})
    }
    return values, len(values) > 0
}
```
Simply moving the switch statement into a helper function would significantly reduce complexity:

```go
func processValue(val string) (name string, defaultValue string) {
	switch {
	case strings.Contains(val, ":?"):
    	name, _ = partition(val, ":?")
	case strings.Contains(val, "?"):
    	name, _ = partition(val, "?")
	case strings.Contains(val, ":-"):
    	name, defaultValue = partition(val, ":-")
	case strings.Contains(val, "-"):
    	name, defaultValue = partition(val, "-")
	default:
    	name = val
	}
	return name, defaultValue
}

func extractVariable(value any, pattern *regexp.Regexp) ([]extractedValue, bool) {
    sValue, ok := value.(string)
    if !ok {
   	 return []extractedValue{}, false
    }
    matches := pattern.FindAllStringSubmatch(sValue, -1)
    if len(matches) == 0 {
   	 return []extractedValue{}, false
    }
    values := []extractedValue{}
    for _, match := range matches {
   	 groups := matchGroups(match, pattern)
   	 if escaped := groups["escaped"]; escaped != "" {
   		 continue
   	 }
   	 val := groups["named"]
   	 if val == "" {
   		 val = groups["braced"]
   	 }
   	 
   	 name, defaultValue := processValue(val)

   	 values = append(values, extractedValue{name: name, value: defaultValue})
    }
    return values, len(values) > 0
}
```
This reduces the CCN of extractVariable from 10 to 6, i.e. 40% reduction.

## importTar (Adam)

```go
func importTar(name string, s Writer, reader io.Reader) error {
    tr := tar.NewReader(&LimitedReader{R: reader, N: maxAllowedFileSizeToImport})
    tlsData := ContextTLSData{
   	 Endpoints: map[string]EndpointTLSData{},
    }
    var importedMetaFile bool
    for {
   	 hdr, err := tr.Next()
   	 if err == io.EOF {
   		 break
   	 }
   	 if err != nil {
   		 return err
   	 }
   	 if hdr.Typeflag != tar.TypeReg {
   		 // skip this entry, only taking files into account
   		 continue
   	 }
   	 if err := isValidFilePath(hdr.Name); err != nil {
   		 return errors.Wrap(err, hdr.Name)
   	 }
   	 if hdr.Name == metaFile {
   		 data, err := io.ReadAll(tr)
   		 if err != nil {
   			 return err
   		 }
   		 meta, err := parseMetadata(data, name)
   		 if err != nil {
   			 return err
   		 }
   		 if err := s.CreateOrUpdate(meta); err != nil {
   			 return err
   		 }
   		 importedMetaFile = true
   	 } else if strings.HasPrefix(hdr.Name, "tls/") {
   		 data, err := io.ReadAll(tr)
   		 if err != nil {
   			 return err
   		 }
   		 if err := importEndpointTLS(&tlsData, hdr.Name, data); err != nil {
   			 return err
   		 }
   	 }
    }
    if !importedMetaFile {
   	 return errdefs.InvalidParameter(errors.New("invalid context: no metadata found"))
    }
    return s.ResetTLSMaterial(name, &tlsData)
}
```

We should be able to reduce complexity by a healthy margin by adding helper functions to perform the actions of the nested if / else if statement.

```go
func processMetaFile(tr *tar.Reader, s Writer, name string) (bool, error) {
    data, err := io.ReadAll(tr)
    if err != nil {
   	 return false, err
    }
    meta, err := parseMetadata(data, name)
    if err != nil {
   	 return false, err
    }
    if err := s.CreateOrUpdate(meta); err != nil {
   	 return false, err
    }
    return true, nil
}

func processTlsFile(tr *tar.Reader, tlsData ContextTLSData, name string) error {
    data, err := io.ReadAll(tr)
    if err != nil {
   	 return err
    }
    if err := importEndpointTLS(&tlsData, name, data); err != nil {
   	 return err
    }
    return nil
}

func importTar(name string, s Writer, reader io.Reader) error {
    tr := tar.NewReader(&LimitedReader{R: reader, N: maxAllowedFileSizeToImport})
    tlsData := ContextTLSData{
   	 Endpoints: map[string]EndpointTLSData{},
    }
    var importedMetaFile bool
    for {
   	 hdr, err := tr.Next()
   	 if err == io.EOF {
   		 break
   	 }
   	 if err != nil {
   		 return err
   	 }
   	 if hdr.Typeflag != tar.TypeReg {
   		 // skip this entry, only taking files into account
   		 continue
   	 }
   	 if err := isValidFilePath(hdr.Name); err != nil {
   		 return errors.Wrap(err, hdr.Name)
   	 }
   	 if hdr.Name == metaFile {
   		 importedMetaFile, err = processMetaFile(tr, s, name)
   	 } else if strings.HasPrefix(hdr.Name, "tls/") {
   		 err = processTlsFile(tr, tlsData, hdr.Name)
   	 }
    }
    if !importedMetaFile {
   	 return errdefs.InvalidParameter(errors.New("invalid context: no metadata found"))
    }
    return s.ResetTLSMaterial(name, &tlsData)
}
```

This reduces the CCN of importTar from 14 to 9, ie. ~36% reduction.
## interactiveExec 

```func interactiveExec(ctx context.Context, dockerCli command.Cli, execConfig *types.ExecConfig, execID string) error {
    // Interactive exec requested.
    var (
   	 out, stderr io.Writer
   	 in      	io.ReadCloser
    )

    if execConfig.AttachStdin {
   	 in = dockerCli.In()
    }
    if execConfig.AttachStdout {
   	 out = dockerCli.Out()
    }
    if execConfig.AttachStderr {
   	 if execConfig.Tty {
   		 stderr = dockerCli.Out()
   	 } else {
   		 stderr = dockerCli.Err()
   	 }
    }
    fillConsoleSize(execConfig, dockerCli)

    client := dockerCli.Client()
    execStartCheck := types.ExecStartCheck{
   	 Tty:     	execConfig.Tty,
   	 ConsoleSize: execConfig.ConsoleSize,
    }
    resp, err := client.ContainerExecAttach(ctx, execID, execStartCheck)
    if err != nil {
   	 return err
    }
    defer resp.Close()

    errCh := make(chan error, 1)

    go func() {
   	 defer close(errCh)
   	 errCh <- func() error {
   		 streamer := hijackedIOStreamer{
   			 streams:  	dockerCli,
   			 inputStream:  in,
   			 outputStream: out,
   			 errorStream:  stderr,
   			 resp:     	resp,
   			 tty:      	execConfig.Tty,
   			 detachKeys:   execConfig.DetachKeys,
   		 }

   		 return streamer.stream(ctx)
   	 }()
    }()

    if execConfig.Tty && dockerCli.In().IsTerminal() {
   	 if err := MonitorTtySize(ctx, dockerCli, execID, true); err != nil {
   		 fmt.Fprintln(dockerCli.Err(), "Error monitoring TTY size:", err)
   	 }
    }

    if err := <-errCh; err != nil {
   	 logrus.Debugf("Error hijack: %s", err)
   	 return err
    }

    return getExecExitStatus(ctx, client, execID)
}
```

We can reduce the complexity of this function by moving the if and if-else statements in the beginning pertaining to IO to a helper function. 

```func prepareStreams(dockerCli command.Cli, execConfig *types.ExecConfig) (io.ReadCloser, io.Writer, io.Writer) {
	var (
		in     io.ReadCloser
		out    io.Writer
		stderr io.Writer
	)

	if execConfig.AttachStdin {
		in = dockerCli.In()
	}
	if execConfig.AttachStdout {
		out = dockerCli.Out()
	}
	if execConfig.AttachStderr {
		if execConfig.Tty {
			stderr = dockerCli.Out()
		} else {
			stderr = dockerCli.Err()
		}
	}

	return in, out, stderr
}
```
The new interactiveExec function would look like this:
```
func interactiveExec(ctx context.Context, dockerCli command.Cli, execConfig *types.ExecConfig, execID string) error {
    // Interactive exec requested.
    in, out, stderr := prepareStreams(dockerCli, execConfig)

    fillConsoleSize(execConfig, dockerCli)

    client := dockerCli.Client()
    execStartCheck := types.ExecStartCheck{
   	 Tty:     	execConfig.Tty,
   	 ConsoleSize: execConfig.ConsoleSize,
    }
    resp, err := client.ContainerExecAttach(ctx, execID, execStartCheck)
    if err != nil {
   	 return err
    }
    defer resp.Close()

    errCh := make(chan error, 1)

    go func() {
   	 defer close(errCh)
   	 errCh <- func() error {
   		 streamer := hijackedIOStreamer{
   			 streams:  	dockerCli,
   			 inputStream:  in,
   			 outputStream: out,
   			 errorStream:  stderr,
   			 resp:     	resp,
   			 tty:      	execConfig.Tty,
   			 detachKeys:   execConfig.DetachKeys,
   		 }

   		 return streamer.stream(ctx)
   	 }()
    }()

    if execConfig.Tty && dockerCli.In().IsTerminal() {
   	 if err := MonitorTtySize(ctx, dockerCli, execID, true); err != nil {
   		 fmt.Fprintln(dockerCli.Err(), "Error monitoring TTY size:", err)
   	 }
    }

    if err := <-errCh; err != nil {
   	 logrus.Debugf("Error hijack: %s", err)
   	 return err
    }

    return getExecExitStatus(ctx, client, execID)
}
```
This would result in a reduction of complexity down to 3 from 10, which is a 70% reduction in complexity.

## prettyPrintInfo (Omar)
```go
func prettyPrintInfo(streams command.Streams, info dockerInfo) error {
	// Only append the platform info if it's not empty, to prevent printing a trailing space.
	if p := info.clientPlatform(); p != "" {
  	  fprintln(streams.Out(), "Client:", p)
	} else {
  	  fprintln(streams.Out(), "Client:")
	}
	if info.ClientInfo != nil {
  	  prettyPrintClientInfo(streams, *info.ClientInfo)
	}
	for _, err := range info.ClientErrors {
  	  fprintln(streams.Err(), "ERROR:", err)
	}

	fprintln(streams.Out())
	fprintln(streams.Out(), "Server:")
	if info.Info != nil {
  	  for _, err := range prettyPrintServerInfo(streams, &info) {
  		  info.ServerErrors = append(info.ServerErrors, err.Error())
  	  }
	}
	for _, err := range info.ServerErrors {
  	  fprintln(streams.Err(), "ERROR:", err)
	}

	if len(info.ServerErrors) > 0 || len(info.ClientErrors) > 0 {
  	  return fmt.Errorf("errors pretty printing info")
	}
	return nil
}
```
The function has `CCN 9` and looking at the code it seems to have two logical components, one responsible for client info and another for server info. One suggestion to reduce the `CCN` is to turn `prettyPrintInfo` to a wrapper function by splitting the two logical units to separate functions that act as entry points for `Client` and `Server` prints respectively. One refactor suggestion could be the following:

```go
func prettyPrintClient(streams command.Streams, info dockerInfo) error {
    // Only append the platform info if it's not empty, to prevent printing a trailing space.
    if p := info.clientPlatform(); p != "" {
   	 fprintln(streams.Out(), "Client:", p)
    } else {
   	 fprintln(streams.Out(), "Client:")
    }
    if info.ClientInfo != nil {
   	 prettyPrintClientInfo(streams, *info.ClientInfo)
    }
    for _, err := range info.ClientErrors {
   	 fprintln(streams.Err(), "ERROR:", err)
    }
    return nil
}

func prettyPrintServer(streams command.Streams, info dockerInfo) error {
    fprintln(streams.Out())
    fprintln(streams.Out(), "Server:")
    if info.Info != nil {
   	 for _, err := range prettyPrintServerInfo(streams, &info) {
   		 info.ServerErrors = append(info.ServerErrors, err.Error())
   	 }
    }
    for _, err := range info.ServerErrors {
   	 fprintln(streams.Err(), "ERROR:", err)
    }

    if len(info.ServerErrors) > 0 || len(info.ClientErrors) > 0 {
   	 return fmt.Errorf("errors pretty printing info")
    }
    return nil
}

func prettyPrintInfo(streams command.Streams, info dockerInfo) error {

    prettyPrintClient(streams, info)
    return prettyPrintServer(streams, info)

}

```
This reduces the `CCN` for `prettyPrintInfo` from 9 to 1 which fulfills the 35% criteria. The `CCN`s for the new functions `prettyPrintClient` and `prettyPrintServer ` are 4 and 6 respectively.  



## newPlugin (Emir)
One way of refactoring the code to provide a lower CC would be to create an error handling function, since the function
covers many different errors, e.g from line 83-100 there are 4 different checks for failing. There could be put in
a method called "checkMetaData" that either sends Errs or OK depending on the outcome, such that the function
NewPlugin keeps its primary function, but has a lower CC. Two additional if checks could be moved to a function called checkCommand, to rid a for loop covering three if statements


# Essence Analysis(Way-of-Working)
The team has settled since the first assignment and though the majority of tools were established early on (and has been used since), the communication and the use of these are now starting to flow a little bit more seamlessly. The key practices are both in place and used in daily workflow by all team members. The main channel for communication is discord, and the main tool is a GitHub repo. All members are using these daily, and the team is adapting the tools to match the assignment. Whenever new tools are introduced, they are quickly adapted to, and no major hiccups have been noted. 

According to the essence standard, the team could possibly classify as Working Well, but this is perhaps a hastily decided state. Due to the short time frame, the team has been more focused on actually working than inspecting the Way-of-Working, and thus it is difficult to judge whether we are achieving The whole team is involved in the inspection and adaptation of the Way-of-Working. Depending on how much a single point determines the team state, the team classifies as Working Well or In Place.



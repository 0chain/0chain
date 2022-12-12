# Unit tests

## Table of Contents

- [Intorduction](#introduction)
- [0chain style](#0chain-style)
- [Writing a test](#writing-a-test)  
- [Mocks](#mocks)
   - [Simple mock example](#simple-mock-example)
   - [vektra mockery](#vektra-mockery)
   - [0chain example](#0chain-example)  

## Introduction

Unit tests importance for an agile and continual integration project cannot
be overemphasised. Unit tests -
1. Confirm that the code works to specification. 
   By performing test that fail when we discover an error in the code.
2. Help developers in finding errors in their own code.
3. Prevent regression in the code. By regularly running all unit tests and
   confirming they pass, changes that break previously written code can be
   avoided.
4. Help with making code changes, by allowing you to debug your code or
   someone else's.
5. Act as documentation on how the code works. 
   In fast moving agile projects they might form the only for the only source of
   documentation.
6. Help find concurrency errors in the code. Golang provides the `-race` option 
   for running unit tests; This options gives an error and provides a report 
   when two goroutines show too much interest in the same memory location 
   around the same time. 
   
Ideally a developer should start writing a unit test before they make start on 
the code changes, then the unit test can be developed alongside their code 
changes. Not only does it help the developer, but it helps anyone 
to review or study your changes.

## 0chain style

Unit tests can be written using different styles. Here we aim to standardise
how to write unit tests in the `0chain` project.

[PR3210](https://github.com/0chain/0chain/pull/321) introduces a new mocks package
to `0chain`. Here we intend to give guidance to using these mocks.

GitHub does not house these mock files, instead they will be auto-generated as
 part of the continuous integration unit test checks. To build the mock files
for yourself run from the `0chain` repository root directory
```shell
make build-mocks
```
These will should be ignored when you commit to `git`.

#### Table-driven tests using subtests

The [Go blog](https://blog.golang.org/subtests#TOC_4.) outlines how to write 
table-driven tests using subtests. We should be using this same style.
A simple tests might look roughly like
```go
func TestSomething(t *testing.T) {
    t.Parallel()
    testCases := []struct {
      name  string
      arg  string
      want string
   }{
      {"test", "p1", "result"},
      {"test2", "p2", "result2"},
      {"nil test",},
   }
   for _, tt := range testCases {
      tt := tt  
      t.Run(tt.name, func(t *testing.T) {
      	 t.Parallel()
         result := something(tt.arg)
         require(t, tt.result, result)
      })
   }
}
```

#### Minimisation

Adopt a minimalistic approach when writing tests. Use just enough setup for your test
to give results. This helps to make the tests more time efficient and easier for the 
reader to follow.

#### Do use nonsense input text

You do not need to use realistic test data unless its necessary. In fact, I would
recommend using nonsense text strings when they have no effect. Giving a hash a value
like`"my hash"`, makes it clear that the function makes does not use this value.
Further, when debugging its clear the source of any hash values encountered.

#### Use require

Use [require](https://pkg.go.dev/github.com/stretchr/testify/require), to report
errors not `assert`. Immediately the test reports an error we want the test stopped; 
This speeds up continuous integration checks that rely on running  unit test.
`Require` does this, while `assert` does not.

#### Avoid comments and log outputs

Do not have comments in finalised unit tests. When viewing unit test results we
mainly look for test pass or failure, In some circumstances unit test coverage.
Comments or log outputs just obscures the information.

By all means use comments or logs while developing code, but remove them in the final draft.

#### Use t.Parallel were appropriate

For the `-race` option to be effective in finding concurrency errors, we need to run
concurrent goroutines. Adding `t.Parallel()` forces all the tests to run in parallel,
this allows the race detector to alert us to concurrency issues between the tests.

`t.Parallel` Should sometimes be avoided, plus care should be taken to avoid confounding 
race errors from the unit test itself.

It can require some judgement to distinguish between object that should be common to
all tests, and those that should be constructed anew for each test. Copy by value
for a new object each test, by pointer to share objects between tests.

When writing unit test think

> Can this function run in a multi goroutine environment?

If so the test should allow the function to prove it can handle 
any concurrency issues.

#### You do not need t.Parallel when testing one use functions

Functions called only once per execution, often initialise 
'logical constant' global objects. Initialising these 'logically constant' objects
involves no concurrency checks, hence no necessity exist for our unit tests these
functions in parallel.

#### Avoid using external data

Wherever possible define test data inside `*.go` files, do not use the less
reliable and efficient external file system. 
Maybe if you need to prove your function can handle various 
types of multimedia file formats, or similar, you might have to consider it, 
but even then try mock it.

#### Keep databases in memory

Use in-memory databases and avoid using the file system. As well as time 
efficiency, database on the file system tend to leave oddly named files
lying around to confuse the user. 

If you have no alternative, just remember to delete the files both before and after the test.
Use a `defer` block to delete the database files after the test 
to handle the case when your tests panics.

#### Do not send real http requests

Mock http requests. You can use the `httptest` package, or [create
an interface](https://www.thegreatcodeadventure.com/mocking-http-requests-in-golang/) 
to support the `http.Clientg.Do` method. If not currently possible then change that.

#### Careful reusing input data between tests

Spot the problem with this test.
```go
func init() {
    globalThing = GlobalThing{}
}

func TestChangesInput(t *testing.T) {
    t.Parallel()
    var inputData = "input data"	
    testCases := []struct {
      name  string
      arg1  string
      arg2  string
      want  string
   }{
      {"test 1", inputData, "arg2, ""result"},
      {"test 2", inputData, "different arg2" ,"result2"},
      {"nil test",},
   }
   for _, tt := range testCases {
      tt := tt  
      t.Run(tt.name, func(t *testing.T) {
      	 t.Parallel()
         result := changesInput(&tt.arg1, tt.arg2, &globalThing)
         require(t, tt.result, result)
      })
   }
}
```
Assuming the test function, `changeInput`, changes the input parameter, then
the tests interfere with each other. The result of the second test to run will 
depend on what happens in the first run test. The arg1 parameters both point to 
same area of memory so in particular running the test with `-race` option will
produce a race error at the point the two `t.Run` goroutines change the 
`tt.args2`.

This can be fixed by defining arg1 separately in both tests

```go
    testCases := []struct {
      name  string
      arg1  string
      arg2  string
      want  string
   }{
      {"test 1",  "input data", "arg2, ""result"},
      {"test 2",  "input data", "different arg2" ,"result2"},
      {"nil test",},
   }
```

Notice that exactly the same kind of concurrency issue can occur with the
`globalThing` parameter to `changeInput`. However `globalThing` should
already be protected against concurrency. Indeed, part of the purpose of the 
unit test can be considered to test `globalThing`'s protection against concurrent 
access.

We should endeavor to separate objets that form part of the test conditions, that 
need to be independent between different test, and objects global to the tests that
should be shared.

## Writing a test

Numerous approaches to writing unit tests can be devised, bellow we describe
an approach that works well. It focuses on maximising code coverage while
minimising setup.
 
Unit tests check a function's outputs match its inputs. Ideally we want as little
setup as we can get away with. How our function works in the global scope
of the project does not interest us; we use integration tests for that. Many 
developers do too much work here, making their unit tests look more like integration
tests.

### Write template TestMyTestFunction

Start off with a basic template

```go
func TestMyTestFunction(t *testing.T) {
    testCases := []struct {
    	name string
   }{
      {""},
   }
   for _, tt := range testCases {
      t.Run(tt.name, func(t *testing.T) {
         myTestFunction()
      })
   }
}
```

### Compile

The step involves getting `TestMyTestFunction` to compile. For example some
input parameters will be required. Just define whatever objects the compiler
asks for. 

You might as well add a variables for `myTestFunction`'s return values.

Remember be minimalistic. You should only need to create default objects
at this point.

### Run

Get the test to run without errors. 

Single step though the function. Make any necessary change to prevent
an error. This will likely involve adding more default objects, and
setting any values needed to avoid returning an error. Remember be
minimalistic.

### Code coverage

Devise test to cover all code paths.

Single step though your function again, and this time focus on which code you
`do not` step through.

##### Skipped blocks

Add code to prevent skipping blocks. The typical situation will be an `if` 
statement. Make whoever changes to the test setup you need to get the code
to pass though skipped blocks.

If we have
```go
  if input.Thing != nil {
  	doSomthing()
  }
```
then add
```go
    input.Thing:  Thing{}
```
to the setup.

##### Error messages

A special case of `skipped blocks`.
```go
if err != nil {
    return err
}
```
A familiar go construct. We should build a test that forces each error
generated. Make a note of them. 

Often there will be unachievable errors. The error conditions can never be
satisfied due to the logic of the situation. You don't need to bother with 
those, just focus on the errors that can be generated on normal execution of
the test function.

##### Skipped loops

`For loops` with empty ranges; We need to give them a range to loop though.

So
```go
  for i, val := range input.List {
      doSomething(i, val)
  }   
```
We need to give input.List some values:
```go
  input.List: []string{"", ""} 
```

##### Branches

```go
if input.path < 10 {
    doSomething()
} else {
    doSomethingElse()
`}
```
When the code branches into two paths, it signifies that we need two
tests. Investigate the conditions that trigger each branch and build a test
for each branch.
```go
{
    name: "use top path"
    input { path: 9}
},
{
    name: "use bottom path"
    input{ path: 10}
}
```

#### External function calls
 
Only our test function needs testing, not any external functions that
it might call. We need to mock these out external function calls. 
Check that our function passes though the correct parameters 
and handles the response correctly.

If the function forms part of an interface then we can use [Mocks](#mocks), 
as in the next section.
```go
  if iFace.Method(7) == 10 {
    everyThinkIsOk()	
  }
```
Then somewhere in the pre-function setup:
```go
    mockIFace.On("Method", 7).Return(10)
```

Otherwise, we should fine a way of handling the external call smoothly. Try to
do the minimal necessary.

Usually unit tests should not have any effect on the production code. However,
one possibility exception might be to make problem methods part of an interface
so that we can mock them.

### Configure test setup

Having done the preliminary steps, we have a pile object definitions, 
and a list of tests we wish to run. However, everything we need should be
already be there. We need to shuffle everything around so all our tests
work. 

1. Create a test for each branch identified in [branches](#branches).
2. Create test entry for each input. For population the direct
    inputs to the function, and the return values for any  mocked out internal calls.
3. Create a `want` test entry for each output. For both the directly returned
    objects, and the input parameters for any mocked out internal calls.
4. Add a local function to set the mock expectations. So for every mocked out
    internal call, create a `mockedObject.On("functionName", inParms).Return(outParms)`
    call for each mocked out internal call. Use the test input and want parters
    to configure the `On` method` calls. The mock boiler code will panic for any mock
    calls that do not match one of the `On` calls.
5. In the `t.Run` block you will want to do at least some of:
    * Call the test function to set mock expectations.
    * Prepare the functions input parameters.
    * Confirm the functions output match the test wants.
    * Call [AssertExpectationsForObjects](https://pkg.go.dev/github.com/stretchr/testify/mock#AssertExpectationsForObjects)
    to confirm the internal mocked out calls proceeded as expected.

Remember that care needs to be taken to avoid sharing memory between tests.
Copy by value not pointer. 

If you feel tempted to set up a working environment for your test
function to work in, then don't. Really don't. Strive to minimise the
amount of setup you need.

Idiot check the function's purpose. You should know by now, but just make sure.
Do your tests align with these purposes.

## Mocks

The test style described in the previous section work well enough for 
simple functions, however our functions can often have input and output data
other than the obvious input arguments and return values. An obvious 
example would be a database object which our test function might access 
using database `get` and `set` functions.

Unit tests should concern themselves only with the code of the `tested 
function`, ideally we want to avoid running functions called internally;
such as the database `get` and `set` of the previous paragraph.

Issues with testing these function might include:
1. Someone has tested them elsewhere. No point in repeating the test.
2. They use complicated runtime objects that setting up 
   would needlessly complicate our test.
3. Extra processing that reduce the time efficiency of the test.

We use Mocks as a solution to this issue. 

### Simple mock example

Imagine we model fredy's walk into town as follows.
```go
type ThingsToDo interface {
	TalkTo(string)
	Buys(string) int
}

func (fredy *Fred) goesToTown (money int, thingsTodo ThingsToDo) int {
    TalkTo("sally")
    money -= Buys("chocolate cake")
    money -= Buys("bus ticket")
    TalkTo("bob")
    money -= Buys("groceries")
    TalkTo("sam")
    money -= Buys("bus ticket")
    TalkTo("sally")
    return money	
}
```
To test `goesToTown` function, but not the `ThingstoDo` interface we use 
[mocks](https://pkg.go.dev/github.com/stretchr/testify/mock). 

A mock object implements an interface and allows you to front load your test with
expectations for your tested function. So here our `goestToTown` method calls 
`TalkTo` and `Buys` four times each, We define a new implementation, 
`mock.ThingsTodo` of the `ThingsToDo` interface that contains a 
`github.com/stretchr/testify/mock.Mock` object.

```go
func TestGoesToTown(t *testing.T) {
	...

    for _, tt := range testCases {
      tt := tt
      t.Run(tt.name, func(t *testing.T) {
      	 t.Parallel()
         thingsToDo = mock.ThingsToDo{}
         fredy = &Fred{}
         
         thigsToDo.On("TalkTo", "sally").Twice()
         thigsToDo,On("TalkTo", "bob").Once()
         thigsToDo.On("TalkTo", "sam").Once()
         thigsToDo.On("Buys", "chocolate cake").Returns(100).Once()
         thigsToDo.On("Buys", "bus ticket").Returns(200).Twice()
         thigsToDo.On("Buys", "groceries").Returns(5000).Once()
         
         money := tt.money
         money = fredy.goesToTown(money, thigsToDo)
         requre.EqualValue(t, moeny, tt.want.money)
         
         require.True(t, mock.AssertExpectationsForObjects(t, thigsTodo))
      }
    }
}
```
The key calls:
```go
   thigsToDo.On("TalkTo", "sally").Twice()
   thigsToDo,On("TalkTo", "bob").Once()
   thigsToDo.On("TalkTo", "sam").Once()
   thigsToDo.On("Buys", "chocolate cake").Returns(100).Once()
   thigsToDo.On("Buys", "bus ticket").Returns(200).Twice()
   thigsToDo.On("Buys", "groceries").Returns(5000).Once()
```
These set up our mock object with expectations of what `goestToTown` will call; 
these can put these in the `t.Run` or earlier when setting each test's data.
At the end of the `t.Run` block call:
```go
   require.True(t, mock.AssertExpectationsForObjects(t, thigsTodo))
```
This Checks that `fredy.goesToTown` met our expectations. 

### Vektra mockery

To help with our unit tests, in `0chain`, we autogenerate mock object for all our
interface objects using the
[vektra/mockery](https://github.com/vektra/mockery) package. 

As we autogenerate these mock object, we avoid keeping them under
version control. Instead, we generate them when we run unit tests during 
continuous integration. Developers can generate them using the `make build-mocks`

To take advantage of autogenerating these `mocks` developers will need to install `vektra mockery` on
their machines with `go get github.com/vektra/mockery/v2/.../` or otherwise
as described in [mockery installation](https://github.com/vektra/mockery#installation).

### 0chain example

`balances cstate.StateContextI`, a key `0chain` interface, allows access to
the blockchain's markle patricia tire blockchain database. Passed in to many
`0chain` methods it can provide hidden input and output parameters. 

`0chain.net/smartcontract/storagesc/block_reward_test.go` provides an example
of handling `balances cstate.StateContextI`. Here we test the storage smart 
contract method `payBlobberBlockRewards`. 

`payBlobberBlockRewards` makes the following `cstae.ContextI` calls:
```go
val, err = balances.GetTrieNode(scConfigKey(ssc.ID))

// for each blobber
val, err = balances.GetTrieNode(stakePoolKey(ssc.ID, blobberID))

// for each stake holder of blobbers receiving a reward
err = balances.AddMint(&state.Mint{
   Minter:     ADDRESS, ToClientID: payment.to, Amount:     payment.amount,
})   
err = balances.AddMint(&state.Mint{
   Minter:     ADDRESS, ToClientID: payment.to,Amount:     payment.amount, 
})

// for each blobber receiving a reward
_, err = balances.InsertTrieNode(stakePoolKey(sscKey, blobberID), sp)
// to record change to minted
_, err = balances.InsertTrieNode(scConfigKey(ssc.ID), conf)
```
In the unit test we define a function `setExpectations`, using pseudo code 
skipping ont all, but the mock calls
```go
var setExpectations = func(t *testing.T, p parameters) (*StorageSmartContract, cstate.StateContextI) {
    var balances = &mocks.StateContextI{}
	...
    balances.On("GetTrieNode", scConfigKey(ssc.ID)).Return(conf, nil).Once()
    for all blobbers stake pools sPool {
      // get the stake pool for each blobber   
      balances.On("GetTrieNode", stakePoolKey(ssc.ID, id)).Return(&sPool, nil).Once()
      ...
      for blobber stakehoders stakehoders {
      	...
      	// mint reward for each stakeholder
         balances.On("AddMint", &state.Mint{
            Minter: ADDRESS, ToClientID: stakehoders, Amount: reward.usage,
         }).Return(nil)
         balances.On("AddMint", &state.Mint{
            Minter: ADDRESS, ToClientID: stakehoders, Amount: reward.capacity,
         }).Return(nil)
      }
      
      // mint reward for each blobber
      balances.On("AddMint", &state.Mint{
        Minter: ADDRESS, ToClientID: blobber, Amount: blobber.usage,
      }).Return(nil)
      }
      balances.On("AddMint", &state.Mint{
        Minter: ADDRESS, ToClientID: blobber, Amount: blobber.capacity,
      }).Return(nil)
   }

   for each blobber stake pool sPool we chnged {
      ...
      // As we changed the stake pool, and the state pool has a map
      // which is copied by pointer, we need to use MatchedBy
      balances.On(
         "InsertTrieNode",
         stakePoolKey(ssc.ID, sPool.Settings.DelegateWallet),
         mock.MatchedBy(func(sp *stakePool) bool {
            ...
            return sp.Rewards.Charge == rewards[i].serviceChargeCapacity+rewards[i].serviceChargeUsage &&
               sp.Rewards.Blobber == rewards[i].total &&
               sp.Settings.DelegateWallet == sPool.Settings.DelegateWallet      
      }),).Return("", nil).Once()
   }	
    
   balances.On("InsertTrieNode", scConfigKey(ssc.ID), conf).Return("", nil).Once()    
}
```
> The `mocks` package tests for equality of the mock object parameters using 
deep equality, which expects pointer values to be equal. 
This can cause a problem in some complicated situations. 
To handle this, the mock package allow you to use your own function 
to determine mock parameter equality. As we see here
```go
balances.On(
   "InsertTrieNode",
   stakePoolKey(ssc.ID, sPool.Settings.DelegateWallet),
   mock.MatchedBy(func(sp *stakePool) bool {
      ...
      return sp.Rewards.Charge == rewards[i].serviceChargeCapacity+rewards[i].serviceChargeUsage &&
         sp.Rewards.Blobber == rewards[i].total &&
         sp.Settings.DelegateWallet == sPool.Settings.DelegateWallet      
}),).Return("", nil).Once()
```
Instead of an `input parameter` we pass `mock.MatchedBy` with our 
personalised equality checker function as a MatchedBy parameter.

The pseudocode structure of the test now looks like
```go
func TestPayBlobberBlockRewards(t *testing.T) {
   type parameters struct { 
    ...
   }
   ...
   var setExpectations = func(t *testing.T, p parameters,) (*StorageSmartContract, cstate.StateContextI) {
        var balances = &mocks.StateContextI{}   
   	    // as above tell balances what calls to expect
   }
   
   type want struct {
      error    bool
      errorMsg string
   }
   tests := []struct {
      name       string
      parameters parameters
      want       want
   }{
      {
         name: "1 blobbers",
         parameters: parameters{
            ...
         },
      },
      ...
   }
   for _, tt := range tests {
      tt := tt
      t.Run(tt.name, func(t *testing.T) {
         t.Parallel()
         // set the expectations for this test.
         ssc, balances := setExpectations(t, tt.parameters)
         
         // call the method being tested
         err := ssc.runTestedFunction(balances)
         
         require.EqualValues(t, tt.want.error, err != nil)
         if err != nil {
            require.EqualValues(t, tt.want.errorMsg, err.Error())
            return
         }
         
         // confirm that the `payBlobberBlockRewards` called
         // the StateContextI interface object as expected.
         require.True(t, mock.AssertExpectationsForObjects(t, balances))
      })
   }
```
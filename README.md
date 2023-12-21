# About

Go Deserializer for Acceptable Safety Side-Engine (or Godasse) is an alternative deserialization mechanism for Go.

# Why?

Go provides the core of a mechanism to deserialize data, but it has several shortcomings.

## No support for missing fields

Most serialization formats (JSON, YAML, XML, etc.) differentiate between a
value that is not part of the message (or fields specified as `undefined`
in JavaScript) and a value that is specified as empty (e.g. `null` or `""`).

By opposition and by design, Go's `encoding/json` does not make the difference between a missing field and a
zero value. While this is _often_ an acceptable choice, there are many cases in which
the default value specified by a protocol is not 0. For instance:

```typescript
{
    "startAfter": number, // A number of milliseconds since the beginning of the game, or now if unspecified.
}
```

While there are patterns that let a developer work around such issues, these patterns are
fairly sophisticated, error-prone and need to be discovered manually, as they appear wholly
undocumented.

By opposition, Godasse:

- supports a simple mechanism to provide default values or constructor for missing or private fields;
- rejects messages with missing field if no default value or constructor has been provided.

## No support for validation
 
When consuming data received from an untrusted source (e.g. the web) or even when working in a team
in which other developers may have made changes to their code without informing everyone, it is important
to validate any data received. This helps both catch potential attacks and help other developers debug
the messages they send.
 
Most (de)serialization mechanisms provide some support for validation. Out of the box, Go's standard
library doesn't.

 
Again, there are patterns that let a developer work around such issues, but they're more complicated,
undocumented and error-prone than they seem.

By opposition, Godasse supports a simple mechanism to provide validation.


# Usage

## Basic Usage

Let's start with a simple request format that we'd like to deserialize from JSON:

```go
type FetchRequest struct {
    Resource string `json:"resource"`
    number   uint8  `json:"number"`
}
```

We can generate a JSON deserializer for this request. Usually, this is something that
you do at startup, as it is going to verify a number of important properties.

```go
func main() {
    options := godasse.deserialize.Options {
        // We want to apply the renamings from tag `json`.
        MainTagName: "json",
    }
    deserializer, err := godasse.deserialize.MakeMapDeserializer[FetchRequest](options)
    if err != nil {
        panic(err)
    }
}
```

...and run this! Oops, we have a panic!
```
    struct FetchRequest contains a field "number" that is not public, you should either make it public or specify an initializer with `Initializer` or `UnmarshalJSON`"
```

Good catch, Godasse! Let's fix that.

```go
type FetchRequest struct {
    Resource string `json:"resource"`
    Number   uint8  `json:"number"`
}
```

Alright, now our code passes!

## Using our deserializer

Let's test it

```go
deserialized, err := deserializer.DeserializeJSONString(`{
    "resource": "/a/b/c",
    "number": 1
}`)

if err != nil {
    panic(err)
}

fmt.Print("We have deserialized ", *deserialized)
```

... and it runs!


## Missing fields

Now, what happens if we forget a field?

```go
deserialized, err := deserializer.DeserializeJSONString(`{
    "resource": "/a/b/c",
}`)

if err != nil {
    panic(err)
}
```

Well, this fails with

```
    missing primitive value at FetchRequestBasic.number, expected uint8
```

If you're using Godasse, that's probably what you expected!

But in our format, we don't want to fail if `number` is unspecified, we'd like to
default to `1`.

## Default values

There are several ways to provide a default value. Let's start with the simplest one.

We'll just amend `FetchRequest` to specify a default value:

```go
type FetchRequest struct {
    Resource string `json:"resource"`
    Number   uint8  `json:"number"` `default:"1"`
}
```

Let's test this:

```go
deserialized, err := deserializer.DeserializeJSONString(`{
    "resource": "/a/b/c",
}`)

if err != nil {
    panic(err)
}

fmt.Print("We have deserialized ", *deserialized)
```

And yes, this displays `Number: 1`!

This also works with more sophisticated patterns:

```go
// Additional options for fetching.
type Options struct {
    // Accept responses that have been generated up to `MaxAgeMS` ms ago.
    //
    // Defaults to 10,000.
    MaxAgeMS uint32 `json:maxAgeMS` `default:"10000"`
}

type AdvancedFetchRequest struct {
    Resource string `json:"resource"`
    Number   uint8  `json:"number"` `default:"1"`

    // Additional options for fetching (optional).
    Options  Options `json:"options"` `default:"{}"`
}
```

In this case, if `options` is missing, it will default to `{}`
*and* its contents are initialized using the default values
specified in `Options`.

```go
deserialized, err := deserializer.DeserializeJSONString(`{
    "resource": "/a/b/c",
}`)

if deserialized.Options.MaxAgeMS != 10000 {
    panic("We should have inserted a default value!")
}
```

The rules for `default` are as follows:

- Godasse **never** injects a default value on your sake;
- for any scalar type (number, strings, booleans), you can specify any value that can be parsed;
- for pointers, the only default value accepted is `nil`;
- for slices and arrays, the only default value accepted is `[]`;
- for structs, the only default value accepted is `{}` (but that shouldn't be a limitation, see above).

Don't worry, if you need something more than that, we have you covered!

## Default constructors

Let's consider a variant format in which instead of `MaxAgeMS`, we have a `MinDateMS`,
which should default to "now minus 10s". For this purpose, we're going to use
tag `orMethod`.


```go
// Additional options for fetching.
type Options struct {
    // Accept responses that have been generated since `MinDateMS`.
    //
    // Defaults to "now minus 10s".
    MinDateMS uint64 `json:minDateMS` `orMethod:"DefaultMinDateMS"`
}

// Compute the default version for `MinDateMS`. Note that this method
// has been attached with `DefaultMinDateMS`.
func (Options) DefaultMinDateMS() (uint64, error) {
    result := time.Now().UnixMilli() - 10000
    return result, nil
}
```

The rules for `orMethod` are as follows:

- Godasse **never** injects a `orMethod` on your sake;
- you cannot have both a `orMethod` and a `default`;
- the `orMethod` must be a method of the same struct;
- the `orMethod` must take 0 arguments and return `(T, error)` where `T` is the type of your field;
- the order in which `orMethod`s is called is unspecified (and actually varies).

Don't worry, Godasse will check these properties when generating the deserializer.

## Initializing private fields

In some cases, you may wish to add private fields to your struct. For instance,
perhaps you wish to have a record of _when_ the request was passed?

The bad news is that tags cannot be attached to private fields (well,
they can, but Go libraries can't see the private fields or tags), so
we can't use `orMethod` or `default`.

For this purpose, Godasse has an interface `Initializer`, which 
can be implemented as such:

```go
type AdvancedFetchRequest struct {
    Resource string `json:"resource"`
    Number   uint8  `json:"number"` `default:"1"`
    Options  Options `json:"options"` `default:"{}"`

    // The instant at which the request was received.
    date     Time
}

func (request* AdvancedFetchRequest) Initialize() error {
    request.date = time.Now()
}

// Double-check that we have implemented Initializer.
var _ godasse.validation.Initializer = &AdvancedFetchRequest{}
```

Now, Godasse will run `Initialize()` to fill in any missing fields,
including private fields.

The rules for `Initializer` are as follows:

- Godasse **never** injects a `Initializer` on your sake;
- `Initialize` must be a method of the same struct;
- `Initialize` must take 0 arguments and return `error`;
- `Initialize` must be implemented on a pointer, rather than a struct (otherwise any change would be lost immediately);
- `Initialize` is called immediately after creating the struct, before parsing the fields.

Again, Godasse will check these rules while creating the deserializer.

## Validating/rewriting data

Last but not least, let's add some validation!

Let's say, perhaps our `Number` should always be between 0 and 100?

```go
func (request *AdvancedFetchRequest) Validate() error {
    if request.number > 100 {
        return fmt.Errorf("Invalid number, expected a value in [0, 100], got %d", request.number)
    }
    // Otherwise, everything is fine.
    return nil
}

// Double-check that we have implemented Validator.
var _ godasse.validation.Validator = &AdvancedFetchRequest{}
```

Now Godasse will run `Validate()` to confirm that everything is valid.

The rules for `Validator` are as follows:

- Godasse **never** injects a `Validator` on your sake;
- `Validate` must be a method of the same struct;
- `Validate` must take 0 arguments and return `error`;
- `Validate` must be implemented on a pointer, rather than a struct;
- `Validate` is called after having parsed all fields;
- `Validate` can modify the structure, if you wish.

# Alternatives

## Making every field a pointer

The recommended workaround to detect missing fields is to:

1. Define every field as a pointer.
2. Write a pass that checks every `nil` field and replaces it with an adequate default value.

This means rewriting our example as follows:

```go
type Options struct {
    MinDateMS *uint64 `json:minDateMS`
}

type AdvancedFetchRequest struct {
    Resource *string `json:"resource"`
    Number   *uint8  `json:"number"`
    Options  *Options `json:"options"`

    // The instant at which the request was received.
    date     *Time
}

func deserialize(data []byte) (*AdvancedFetchRequest, error) {
    result := new(AdvancedFetchRequest)

    err := json.Unmarshal(data, result)
    if err != nil {
        // FIXME: Presumably add some context
        return nil, err
    }

    // Check for missing fields.
    if result.Resource == nil {
        return nil, fmt.ErrorF("in AdvancedFetchRequest, field `resource` should be specified")
    }
    if result.Number == nil {
        return nil, fmt.ErrorF("in AdvancedFetchRequest, field `number` should be specified")
    }
    if result.Options == nil {
        result.Options = &Options {}
    }

    // Check for missing fields within fields.
    if result.Options.MinDateMS == nil {
        result.Options.MinDateMS = time.Now().UnixMilli() - 10000
    }

    result.date = &time.Time()

    // Perform validation.
    if result.Number > 100 {
        return nil, fmt.Errorf("invalid number, expected a value in [0, 100], got %d", result.Number)
    }

    // Perform validation within fields.
    // (in this case, nothing to do)

    return result, nil
}
```

Pros of this approach:

- No dependency.

Cons of this approach:

- Very easy to miss checking one field and end up with a crash in production.
- A bit awkward if the same `struct` is shared between several messages.
- Downstream code now needs to deals with pointers, even if pointers are not the appropriate data structure.
- More verbose.
- No detection of errors at startup.
- Less precise error messages.
- You don't get to use a module called "godasse".


## Implementing Unmarshaler

With a little elbow grease, Go supports initialization and validation with
`Unmarshal`.

Let us start with our example

```go
// Additional options for fetching.
type Options struct {
    // Accept responses that have been generated since `MinDateMS`.
    //
    // Defaults to "now minus 10s".
    MinDateMS uint64 `json:minDateMS`
}


type AdvancedFetchRequest struct {
    Resource string `json:"resource"`
    Number   uint8  `json:"number"`
    Options  Options `json:"options"`

    // The instant at which the request was received.
    date     Time
}
```

Now, we can implement `UnmarshalJSON` for `Options` to setup default values:

```go
// Critically, implement it on `*Options`, not on `Options`.
func (dest *Options) UnmarshalJSON(buf []byte) error {
    // Pre-initialize fields.
    dest.MinDateMS = time.Now().UnixMilli() - 10000
 
    // Perform deserialization.
    err := json.Unmarshal(buf, result)
    if err != nil {
        // TODO: Presumably, add some context.
        return err
    }
    return nil
}
```

That's... not too bad. A bit repetitive and error-prone but we can live with that.

Now, what about `AdvancedFetchRequest`? Ah, well, there it's a bit more complicated
because we want the ability to detect whether a field is missing:

```go
func (dest *AdvancedFetchRequest) UnmarshalJSON(buf []byte) error {
    // The same type as `AdvancedFetchRequest`, except every field is a pointer.
    type Aux struct {
        Resource *string
        Number   *uint8
        Options  *Options
    }
    aux := new(Aux)

    // First, unmarshal to the pointerized type.
    err := json.Unmarshal(aux)
    if err != nil {
        // TODO: Presumably, add some context.
        return err
    }

    // Reject nil fields.
    if aux.Resource == nil {
        return fmt.ErrorF("in AdvancedFetchRequest, field `resource` should be specified")
    }
    resource := *aux.Resource

    if aux.Number == nil {
        return fmt.ErrorF("in AdvancedFetchRequest, field `Number` should be specified")
    }
    number := *aux.Number

    // Or inject default values.
    if aux.Options == nil {
        err = json.Unmarshal("{}", aux.Options)
        if err != nil {
            // Wait, how could his happen?
            return err
        }
    }
    options := *aux.Options

    // Inject default values for private fields.
    time := time.Now()

    // Reconstruct destination field.
    dest.Resource = resource
    dest.Number = number
    dest.Options = options
    dest.time = time

    // Perform validation.
    if dest.Number > 100 {
        return fmt.Errorf("invalid number, expected a value in [0, 100], got %d", request.number)
    }

    // Finally, we should be good.
    return nil
}
```

Pros of this approach:

- No dependency.
- You get to write more Go code!
- You get to spend more time debugging Go code!
- You get to spend more time debugging Go code *in production*!
- You get to write more tests!

Cons of this approach:

- For some reason, I couldn't find any documentation on this approach.
- Only works with JSON.
- More verbose.
- More error-prone.
- No detection of errors at startup.
- Less precise error messages.
- You don't get to use a module called "godasse".

## Json schema

(To be Documented)

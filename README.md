# Word and Lexeme Generator

Small tool used to generate lexemes and words for naming languages and protolangs.

## Usage

```wordgen CONFIG [-count=INT] [-seed=INT] [-output=PATH]```

### Config

`wordgen` requires a language config to function. The config is specified as a `.toml` file in the following format:
```
name = "Language Name"
word = "Word Pattern"

[productions]
prod1 = "pattern"
prod2 = "pattern"
```

The tool will then generate `-count` words using the `CONFIG.word` pattern. If the pattern requires a more complex
pattern (ex. word made from syllables which are in turn made from consonants and vowels), extra intermediate
productions can be specified and referenced as `$production`.

See the [example config](./test.toml) for further info.

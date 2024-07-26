![COMMI Cover](_media/cover.jpg)

# COMMI

COMMI is your git commit friend.

## Usage

After installation, you can use AICommit by simply running:

```bash
commi
```

```bash
commi "fixed CR-22 ticket"
```

### Optional Arguments

- `[subject]`: Specify a subject for the commit message (optional).
- `-a, --auto`: Automatically commit without opening a dialog.
- `-v, --version`: Display version information.

## Configuration

For Anthropic:
```bash
export ANTHROPIC_API_KEY=your_api_key_here
```

For OpenAI:
```bash
export OPENAI_API_KEY=your_api_key_here
```

## Environment Variables

- `ANTHROPIC_API_KEY`: Your Anthropic API key
- `OPENAI_API_KEY`: Your OpenAI API key

## License

Code is released under the MIT License. 

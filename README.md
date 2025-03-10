![COMMI Cover](_media/cover.jpg)

# COMMI

COMMI is your git commit friend.

## Installation

### From Source

To install COMMI from source:

```bash
# Clone the repository
git clone https://github.com/user/commi.git
cd commi

# Build and install
go build
go install
```

## Usage

After installation, you can use AICommit by simply running:

```bash
commi
```

Or if you want to specify a subject for the commit message:

```bash
commi "fixed CR-22 ticket"
```

![COMMI Screenshot 1](_media/screenshot1.png)

![COMMI Screenshot 2](_media/screenshot2.png)

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

## Special thanks to
Sonnet 3.5
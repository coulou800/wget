# wget

This project is a Go implementation of the popular command-line tool `wget`. It allows users to download files from the web, mirror websites, and manage downloads with features like rate limiting and background processing.

## Features

- **File Downloading**: Download files from specified URLs.
- **Rate Limiting**: Control the download speed using options like `400k` or `2M`.
- **Background Downloads**: Run downloads in the background without blocking the terminal.
- **Site Mirroring**: Download entire websites and adjust internal links for offline navigation.
- **Progress Bar**: Visual feedback during downloads using a progress bar.
- **Content-Disposition Handling**: Automatically determine the filename from the server's response.

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/coulou800/wget.git
   cd wget
   ```

2. **Build the project**:
   ```bash
   go build -v
   ```

3. **Run the program**:
   ```bash
   ./wget [options] [URL]
   ```

## Usage

### Basic Command

To download a file:
```bash
./wget https://example.com/file.zip
```

### Rate Limiting

To limit the download speed:
```bash
./wget --rate-limit 400k https://example.com/file.zip
```

### Background Download

To run the download in the background:
```bash
./wget --background https://example.com/file.zip
```

### Site Mirroring

To mirror a website:
```bash
./wget --mirror https://example.com
```

### Input File

To download multiple files from a list:
```bash
./wget -i filename.txt
```

## Flags

- `-O`: Specify a different name for the downloaded file.
- `-P`: Specify the directory to save the downloaded file.
- `--rate-limit`: Limit the download speed (e.g., `400k`, `2M`).
- `-B`: Download the file in the background.
- `-i`: Input file containing URLs to download.
- `--mirror`: Enables site mirroring.

## Logging

The application logs its operations to a file named `wget-log`. This log includes details about the start and end times of downloads, the URLs accessed, and the status of each request.

## License

This project is licensed under the GNU General Public License v3.0. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- This project uses the `mpb` library for progress bars.
- Thanks to the Go community for their support and resources.

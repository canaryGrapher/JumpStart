// jumpstart-ocr: prints the text macOS Vision recognizes in an image.
// Bundled into JumpStart.app/Contents/MacOS by scripts/build-ocr-helper.sh
// and run by internal/ocr for the "Vision" engine. Runs entirely on-device.
import Foundation
import ImageIO
import Vision

func fail(_ message: String, _ code: Int32 = 1) -> Never {
    FileHandle.standardError.write((message + "\n").data(using: .utf8)!)
    exit(code)
}

let args = CommandLine.arguments
guard args.count == 2 else { fail("usage: jumpstart-ocr <image>", 2) }

let url = URL(fileURLWithPath: args[1])
guard let source = CGImageSourceCreateWithURL(url as CFURL, nil),
      let image = CGImageSourceCreateImageAtIndex(source, 0, nil) else {
    fail("cannot read image")
}

let request = VNRecognizeTextRequest()
request.recognitionLevel = .accurate
request.usesLanguageCorrection = true
if #available(macOS 13, *) {
    request.automaticallyDetectsLanguage = true
}

do {
    try VNImageRequestHandler(cgImage: image, options: [:]).perform([request])
} catch {
    fail("recognition failed: \(error.localizedDescription)")
}

let lines = (request.results ?? []).compactMap { $0.topCandidates(1).first?.string }
print(lines.joined(separator: "\n"))

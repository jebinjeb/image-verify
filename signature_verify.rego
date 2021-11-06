package signature

default verified = false
# send HTTP POST request to cosign-wrapper

verified {
    # read the `image` from the `input` that will be verified
    body := { "image": input.image }
    
    # hardcoded consts
    headers_json := { "Content-Type": "application/json" }
    cosignHTTPWrapperURL := "http://localhost:8080/verify"

    # send HTTP POST request to cosign-wrapper
    output := http.send({"method": "post", "url": cosignHTTPWrapperURL, "headers": headers_json, "body": body})
    
    # check if result verified
    output.body.verified
}

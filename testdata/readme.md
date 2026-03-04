# Test Go Files
These files are fed to `gouse` to test it. Each file describes what it tests in
godoc-like comments for `main()`. All their relations are described below.
* * `not_used.{input|golden}` and `used.{input|golden}` cover every first and
    every second processing of the same file correspondingly when variables
    aren’t used. They test the general use of `gouse`.
  * `not_used_{no_provider|var_and_import|var_forms|control_headers|
    switch_forms|unicode}.{input|golden}` test cases when imports are unused
    or missing, when additional declaration sites need fake usages, and when
    current switch and Unicode edge cases are being characterized.
  * `used_gofmted{|_different_name_length|_var_forms|_control_headers|
    _unicode}.{input|golden}` checks cases when files are `gofmt`ed after
    creating fake usages.

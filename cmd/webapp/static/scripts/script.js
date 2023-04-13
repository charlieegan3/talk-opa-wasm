import opa from '/static/scripts/opa-wasm-browser.esm.js';

export { reloadPolicy, validateForm, ready };

var policyBytes = null;
var policyLoadedAt = null;

function ready(fn) {
    if (document.readyState !== 'loading') {
        fn();
    } else {
        document.addEventListener('DOMContentLoaded', fn);
    }
}

function reloadPolicy() {
    fetch('http://localhost:8080/bundles/application/form.wasm').then(response =>
        response.arrayBuffer()
    ).then(bytes => {
        policyBytes = bytes;
        document.getElementById("policy-loaded-at").innerHTML = new Date().toLocaleTimeString();
    })
}

function validateForm(e) {
    e.preventDefault();
    clearFormMessages();

    const formData = new FormData(e.target);

    const inputData = {
        name: formData.get('name'),
        email: formData.get('email'),
        departure_station: formData.get('departure_station'),
        destination_station: formData.get('destination_station'),
        passenger_count: parseInt(formData.get('passenger_count'), 10),
        seats: formData.getAll('seat'),
    }

    const callback = () => {
        fetch('http://localhost:8081/book', {
            method: 'POST',
            body: JSON.stringify(inputData),
            headers: {
                'Content-Type': 'application/json'
            }
        }).then(response => {
            if (response.status === 200) {
                setResult('OK');
            } else if (response.status === 400) {
                response.json().then(data => {
                    showFormMessages("server", data)
                    setResult('ERROR: failed server validation');
                })
            } else {
                setResult('ERROR: unexpected response code');
            }
        }).catch(error => {
            console.error(error);
            setResult('ERROR: request failed');
        })
    }


    if (policyBytes !== null) {
        // validate the form using opa wasm
        opa.loadPolicy(policyBytes).then(policy => {
            const resultSet = policy.evaluate(inputData);
            if (resultSet == null) {
                setResult("ERROR: evaluation")
                console.error("evaluation error");
                return
            }
            if (resultSet.length === 0) {
                setResult("ERROR: result undefined")
                console.log("result undefined");
                return
            }

            const result = resultSet[0].result;

            if (result.length === 0) {
                callback()

                return
            }

            setResult("ERROR: failed local validation")
            showFormMessages("wasm", result)
        })
    } else {
        // ask the server to validate the form
        callback()
    }
}

// unexported
function clearFormMessages() {
    // clear result
    const elem = document.getElementById('result');
    elem.innerHTML = '';

    // clear errors
    const elems = document.querySelectorAll('.error-message');
    for (let i = 0; i < elems.length; i++) {
        elems[i].innerHTML = '';
    }
}

function showFormMessages(source, result) {
    var messagesForFields = {};
    for (let i = 0; i < result.length; i++) {
        if (result[i].field in messagesForFields) {
            messagesForFields[result[i].field].push(result[i].message);
        } else {
            messagesForFields[result[i].field] = [result[i].message];
        }
    }

    for (let field in messagesForFields) {
        setError(field, source + ': ' + messagesForFields[field].join('<br>'));
    }
}

function setError(field, msg) {
    const elem = document.querySelector('.error-message.error-message-' + field);
    elem.innerHTML = msg;
}

function setResult(msg) {
    const elem = document.getElementById('result');
    elem.innerHTML = msg;
}

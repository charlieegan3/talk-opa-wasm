import opa from '/static/scripts/opa-wasm-browser.esm.js';

export { reloadPolicy, validateForm, ready };

var policyBytes = null;

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
    ).then(bytes =>
        policyBytes = bytes
    )
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

    console.log(inputData.seats);

    opa.loadPolicy(policyBytes).then(policy => {
        const resultSet = policy.evaluate(inputData);
        if (resultSet == null) {
            console.error("evaluation error");
            return
        }
        if (resultSet.length === 0) {
            console.log("result undefined");
            return
        }

        const result = resultSet[0].result;

        if (result.length === 0) {
            setResult('OK');
            return
        }

        var messagesForFields = {};
        for (let i = 0; i < result.length; i++) {
            if (result[i].field in messagesForFields) {
                messagesForFields[result[i].field].push(result[i].message);
            } else {
                messagesForFields[result[i].field] = [result[i].message];
            }
        }

        for (let field in messagesForFields) {
            setError(field, messagesForFields[field].join('<br>'));
        }
    })
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

function setError(field, msg) {
    const elem = document.querySelector('.error-message.error-message-' + field);
    elem.innerHTML = msg;
}

function setResult(msg) {
    const elem = document.getElementById('result');
    elem.innerHTML = msg;
}

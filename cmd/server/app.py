import os
import re
import json
import datetime
from pathlib import Path

from flask import Flask, render_template, redirect, request, Response
import requests
from opa_wasm import OPAPolicy

POLICY_LOCATION = Path(__file__).parent / "policy.wasm"

policy_loaded_at = None
bookings = []

if os.path.exists(POLICY_LOCATION):
    os.remove(POLICY_LOCATION)

app = Flask(__name__)


@app.route('/', methods=['GET'])
def index():
    reversed_bookings = bookings
    reversed_bookings.reverse()
    return render_template(
        'index.html',
        policy_loaded_at=policy_loaded_at,
        bookings=reversed_bookings,
    )


@app.route('/book', methods=['POST', 'OPTIONS'])
def book():
    global bookings

    resp = Response()
    resp.headers['Access-Control-Allow-Origin'] = '*'
    resp.headers['Access-Control-Allow-Methods'] = 'POST, OPTIONS'
    resp.headers['Access-Control-Allow-Headers'] = 'Content-Type'

    if request.method == 'OPTIONS':
        return resp

    booking = request.json

    # we only evaluate the policy if it exists to allow demoing
    if os.path.exists(POLICY_LOCATION):
        policy = OPAPolicy(POLICY_LOCATION, builtins={'sprintf': sprintf})

        policy.set_data({"company_name": "ACME"})

        result = policy.evaluate(booking)[0]['result']

        if len(result) > 0:
            resp.headers['Content-Type'] = 'application/json'
            resp.data = json.dumps(result)
            resp.status_code = 400
            return resp

    bookings.append(booking)

    return resp


@app.route('/reload', methods=['GET'])
def reload():
    global policy_loaded_at
    try:
        response = requests.get(
            url="http://localhost:8080/bundles/application/form.wasm",
        )

        f = open(POLICY_LOCATION, "wb")
        f.write(response.content)
        f.close()

        policy_loaded_at = datetime.datetime.now().strftime("%H:%M:%S")

    except requests.exceptions.RequestException:
        return f'HTTP Request failed'

    return redirect('/')


def sprintf(str, *args):
    return re.sub("%\\w", "{}", str).format(*args[0])
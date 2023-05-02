import React, { useState, useRef } from "react";

import './App.css';

function App() {
  // inputDate stores the annotations for a single field
  const [inputData, setinputData] = useState([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [fieldLength, setFieldLength] = useState(0);

  // current clause text
  const [currentClauseText, setCurrentClauseText] = useState('')
  const [currentFieldTitle, setcurrentFieldTitle] = useState('')
  const [currentResultText, setCurrentResultText] = useState([])
  const [selectedRating, setSelectedRating] = useState('')
  const [fieldFound, setFieldFound] = useState('')
  const [temp, setTemp] = useState('1')
  const [numruns, setNumruns] = useState('1')

  // run button has been clicked
  const [runClicked, setRunClicked] = useState(false);


  // refs
  const getFieldRef = useRef("")
  const gotoRef = useRef()
  const promptRef = useRef()
  const clauseRef = useRef()
  const notesRef = useRef()

  // handleGetFieldClick fetches the annotations of the given field
  // id from the backend
  async function handleGetFieldClick() {
    const fieldID = getFieldRef.current.value
    const jsonFieldID = { "id": fieldID }
    const fieldData = await fetch('http://localhost:4000/api/getField', {
      method: "POST",
      headers: {
        'Content-type': 'application/json'
      },
      body: JSON.stringify(jsonFieldID)
    })
      .then((result) => result.json())
    // if field id isn't found display a message
    if (fieldData.title === "field not found!") {
      setFieldFound("field not found!")
    } else {
      setFieldFound("")
      setinputData(fieldData)
      setCurrentClauseText(fieldData.annotations[currentIndex])
      setFieldLength(fieldData.annotations.length)
      setcurrentFieldTitle(fieldData.title)
    }
  }

  // handleTextInputDocID handles the text input which can be used to
  // navigate to a Snippet with given docID
  function handleGotoClick() {
    const indexValue = parseInt(gotoRef.current.value)
    // check the input value is within bounds
    if (isNaN(indexValue)) {
      return
    } else if (indexValue < 1) {
      return
    } else if (indexValue > inputData.annotations.length) {
      return
    } else {
      setCurrentIndex(indexValue - 1)
      setCurrentClauseText(inputData.annotations[indexValue - 1])
    }
    gotoRef.current.value = null
  }

  // handleRunClick takes the current prompt and clause and
  // sends them to the backend for processing by chatGPT
  async function handleRunClick() {
    setRunClicked(true);
    const prompt = promptRef.current.value
    const clause = clauseRef.current.value
    const jsonData = { "prompt": prompt, "clause": clause, "temp": temp, "numruns": numruns }
    // setCurrentResultText("Loading...")
    const gptOutput = await fetch('http://localhost:4000/api/run', {
      method: "POST",
      headers: {
        'Content-type': 'application/json'
      },
      body: JSON.stringify(jsonData)
    })
      .then((result) => result.json())
    setCurrentResultText(gptOutput.gptoutputs)
    setRunClicked(false);
    for (let i = 0; i < gptOutput.gptoutputs.length; i++) {
      console.log(gptOutput.gptoutputs[i])
    }
  }


  // handleSaveClick sends the current state of the app to the backend
  // where it is logged
  async function handleSaveClick() {
    const prompt = promptRef.current.value
    const clause = clauseRef.current.value
    const notes = notesRef.current.value
    const jsonData = { "prompt": prompt, "clause": clause, "result": currentResultText, "notes": notes, "rating": selectedRating, "temp": temp }
    await fetch('http://localhost:4000/api/save', {
      method: "POST",
      headers: {
        'Content-type': 'application/json'
      },
      body: JSON.stringify(jsonData)
    })
      .then((result) => result.json())
  }


  // handleSaveClick sends the current state of the app to the backend
  // where it is logged
  async function handleSaveAndContinueClick() {
    const prompt = promptRef.current.value
    const clause = clauseRef.current.value
    const notes = notesRef.current.value
    const jsonData = { "prompt": prompt, "clause": clause, "result": currentResultText, "notes": notes, "rating": selectedRating, "temp": temp }
    await fetch('http://localhost:4000/api/save', {
      method: "POST",
      headers: {
        'Content-type': 'application/json'
      },
      body: JSON.stringify(jsonData)
    })
      .then((result) => result.json())
    setCurrentResultText("")
    // don't let the index go out of bounds
    if (!((currentIndex + 1) > inputData.length - 1)) {
      setCurrentIndex(currentIndex + 1)
      setCurrentClauseText(inputData.annotations[currentIndex + 1])
    }
  }

  function handleGoBackClick() {
    setCurrentResultText("")
    // don't let the index go out of bounds
    if (!((currentIndex) < 1)) {
      setCurrentIndex(currentIndex - 1)
      setCurrentClauseText(inputData.annotations[currentIndex - 1])
    }
    return
  }

  function handleGoForwardClick() {
    setCurrentResultText("")
    // don't let the index go out of bounds
    if (!((currentIndex + 1) > inputData.annotations.length - 1)) {
      setCurrentIndex(currentIndex + 1);
      setCurrentClauseText(inputData.annotations[currentIndex + 1])
    }
    return
  }

  return (
    <div className="App">
      <div className="row">
        <h1>ChatGPT Tool</h1>

        {/* Fetch field which we want to work with with field id */}
        <label htmlFor="getField">Get Clauses</label>
        <div className="display-flex">
          <input ref={getFieldRef} type="text" name="getField" id="getField" placeholder="Enter Field ID" />
          <button onClick={handleGetFieldClick}> Get Field </button>
          <div>{fieldFound}</div>
        </div>
        {!!fieldLength && (
          <>
            {/* Go to specific clause in field */}
            <label htmlFor="goto">Go to clause</label>
            <div className="display-flex">
              <input ref={gotoRef} type="text" name="goto" id="goto" placeholder="Enter clause index" />
              <button onClick={handleGotoClick}>Go to index</button>
              <button type="submit" onClick={handleGoBackClick}>&lt; Previous clause</button>
              <button type="submit" onClick={handleGoForwardClick}>Next clause &gt;</button>
            </div>

            <div>
              {getFieldRef.current.value && (<p className="field-heading"><u>Current Field:</u> {getFieldRef.current.value + ":"} {currentFieldTitle}</p>)}
              {fieldLength > 0 && (<p className="field-heading"><u>Current Clause:</u> {currentIndex + 1} of {fieldLength}</p>)}
              {/* {`Navigation - Current Field: ${getFieldRef.current.value}, Current Clause: ${currentIndex + 1} of ${fieldLength}`} */}
            </div>

            {/* Prompt */}
            <div>
              <label htmlFor="prompt"> Prompt</label>
              <textarea type="text" name="prompt" id="prompt" ref={promptRef}></textarea>
            </div>

            {/* Temperature */}
            <div>
              <label htmlFor="temperature"> Temperature: {temp} </label>
              <input type="range" id="temperature" name="temperature" defaultValue="1" min="0" max="2" step="0.1" onChange={(e) => setTemp(e.target.value)} />
            </div>

            {/* Current Clause */}
            <div>
              <label htmlFor="clause"> Current Clause</label>
              <textarea name="clause" id="clause" value={currentClauseText} onChange={(e) => setCurrentClauseText(e.target.value)} ref={clauseRef}></textarea>
            </div>

            {/* Number of runs button */}
            <label htmlFor="numruns">Number of runs</label>
            <select id="numruns" name="numruns" onChange={(e) => setNumruns(e.target.value)}>
              <option value="1">1</option>
              <option value="2">2</option>
              <option value="3">3</option>
              <option value="4">4</option>
              <option value="5">5</option>
            </select>

            {/* Run button */}
            <button type="submit" onClick={handleRunClick}>Run</button>
          </>
        )}

        {currentResultText.length >= 1 ? (
          <>
            {/* Result */}
            <div>
              <h2>Result</h2>
              {currentResultText.map((result, i) => {
                return (
                  <div className="result-container" key={`result-${i}`}>
                    <div>
                      {result}
                    </div>
                  </div>
                )
              })}

              {/* Rating */}
              <div>
                <fieldset>
                  <legend>How useful is this?</legend>
                  <div className="display-flex">
                    <div className="radio-option">
                      <label htmlFor="good"> good </label>
                      <input type="radio" name="rating" id="good" value="good" onChange={(e) => setSelectedRating(e.target.value)} />
                    </div>
                    <div className="radio-option">
                      <label htmlFor="meh"> meh </label>
                      <input type="radio" name="rating" id="meh" value="meh" onChange={(e) => setSelectedRating(e.target.value)} />
                    </div>
                    <div className="radio-option">
                      <label htmlFor="bad"> bad </label>
                      <input type="radio" name="rating" id="bad" value="bad" onChange={(e) => setSelectedRating(e.target.value)} />
                    </div>
                  </div>
                </fieldset>
              </div>
            </div>

            {/* Notes */}
            <div>
              <label htmlFor="notes"> Notes</label>
              <textarea ref={notesRef} name="notes" id="notes"></textarea>
            </div>

            {/*Save and Save and Continue buttons */}
            <button type="submit" onClick={handleSaveClick}>Save</button>
            <button type="submit" onClick={handleSaveAndContinueClick}>Save and Continue</button>
          </>
        ) : currentResultText.length === 0 && runClicked === true && <p id="loading-text">Loading Results...</p>}
      </div>
    </div >
  );
}

export default App;

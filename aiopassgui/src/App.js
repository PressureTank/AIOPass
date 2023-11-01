import React, { useState, useEffect } from "react";
import "./App.css";

function App() {
    const [input, setInput] = useState("");
    const [response, setResponse] = useState("");
    const [templates, setTemplates] = useState([]);
    const [selectedTemplate, setSelectedTemplate] = useState(null);

    useEffect(() => {
        // Fetch templates from the backend when the component mounts
        fetchTemplates();
    }, []); // Empty dependency array ensures the effect runs once after the initial render

    const fetchTemplates = async () => {
        try {
            const res = await fetch("http://localhost:8080/templates");
            const data = await res.json();
            setTemplates(data);
        } catch (error) {
            console.error("Error fetching templates:", error);
        }
    };

    const handleTemplateChange = (e) => {
        const templateId = parseInt(e.target.value);
        const selectedTemplate = templates.find(
            (template) => template.id === templateId
        );
        setSelectedTemplate(selectedTemplate);
    };

    const askGPT = async () => {
        let prompt = input;
        if (selectedTemplate) {
            prompt = selectedTemplate.prompt;
        }

        try {
            const res = await fetch("http://localhost:8080/ask", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ prompt }),
            });
            const data = await res.json();
            setResponse(data);
        } catch (error) {
            console.error("Error:", error);
        }
    };

    const deleteTemplate = async () => {
        if (selectedTemplate) {
            try {
                await fetch(
                    `http://localhost:8080/templates/${selectedTemplate.id}`,
                    {
                        method: "DELETE",
                    }
                );
                // Refresh templates after deletion
                fetchTemplates();
                setSelectedTemplate(null);
            } catch (error) {
                console.error("Error deleting template:", error);
            }
        }
    };

    const copyTemplate = async () => {
        if (selectedTemplate) {
            try {
                await fetch("http://localhost:8080/templates", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify({ prompt: selectedTemplate.prompt }),
                });
                // Refresh templates after copying
                fetchTemplates();
            } catch (error) {
                console.error("Error copying template:", error);
            }
        }
    };

    return (
        <div className="App">
            <select onChange={handleTemplateChange}>
                <option value="">Select a Template</option>
                {templates.map((template) => (
                    <option key={template.id} value={template.id}>
                        {template.prompt}
                    </option>
                ))}
            </select>
            <button onClick={copyTemplate}>Copy</button>
            <button onClick={deleteTemplate}>Delete</button>
            <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Or enter your question..."
            />
            <button onClick={askGPT}>Ask</button>
            <div className="response">{response}</div>
        </div>
    );
}

export default App;
